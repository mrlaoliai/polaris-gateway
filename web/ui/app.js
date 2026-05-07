const { createApp, ref, computed, onMounted } = Vue;

createApp({
    setup() {
        const currentTab = ref('dashboard');
        const apiData = ref([]);
        const availableAccounts = ref([]);
        const selectedAccount = ref('all');
        const activePreset = ref('today');
        const concurrency = ref({ active: 0, waiting: 0, max: 0 });
        const startDate = ref('');
        const endDate = ref('');
        let fpInstance = null;

        const logsText = ref('Loading logs...');
        const isAutoScroll = ref(true);

        const nodes = ref([]);
        const settings = ref({
            listen_addr: '127.0.0.1:28888',
            breaker: {
                initial_cooldown_seconds: 60,
                max_cooldown_seconds: 3600,
                failure_threshold: 3,
                failure_window_seconds: 120
            }
        });
        
        const logLevelFilter = ref('all');
        const toast = ref({ show: false, message: '', type: 'success' });
        const showToast = (msg, type = 'success') => {
            toast.value = { show: true, message: msg, type };
            setTimeout(() => { toast.value.show = false }, 3000);
        };

        const nodeModal = ref({ show: false, isEdit: false });
        const nodeForm = ref({
            id: 0, platform: 'openai', name: '', key: '', project_id: '', location: 'global', base_url: '',
            priority: 0, cutoff_percent: 95.0, budget: 0.0, billing_start_date: '2000-01-01', is_enabled: true
        });

        const formatNum = (num) => Number(num).toFixed(4);
        const formatToken = (num) => new Intl.NumberFormat().format(num);
        const formatShortDate = (dt) => dt ? dt.split(' ')[0] : '-';
        const successRateColor = (rate) => rate > 95 ? 'border-emerald-500' : (rate > 80 ? 'border-yellow-500' : 'border-red-500');

        const selectedAccountLabel = computed(() => {
            if (selectedAccount.value === 'all') return '全部汇总';
            const matched = availableAccounts.value.find(a => a.value === selectedAccount.value);
            return matched ? matched.label : selectedAccount.value;
        });

        const groupedApiData = computed(() => {
            const map = {};
            apiData.value.forEach(r => {
                const key = r.account;
                if (!map[key]) {
                    map[key] = {
                        account: r.account, platforms: new Set(), budget: r.budget, cutoff_percent: r.cutoff_percent,
                        start_date: r.start_date, total_cost_usd: r.total_cost_usd, cycle_cost_usd: r.cycle_cost_usd,
                        period_cost_usd: 0, prompt_tokens: 0, completion_tokens: 0, success_count: 0, error_count: 0, breakdown: []
                    };
                }
                map[key].platforms.add(r.platform);
                map[key].period_cost_usd += r.period_cost_usd;
                map[key].prompt_tokens += r.prompt_tokens;
                map[key].completion_tokens += r.completion_tokens;
                map[key].success_count += r.success_count;
                map[key].error_count += r.error_count;
            });
            const result = Object.values(map);
            result.forEach(acc => acc.platforms = Array.from(acc.platforms));
            return result.sort((a,b) => b.period_cost_usd - a.period_cost_usd);
        });

        const singleAccountDetails = computed(() => {
            if (selectedAccount.value === 'all') return [];
            const details = apiData.value.filter(d => d.account === selectedAccount.value);
            return details.sort((a, b) => b.period_cost_usd - a.period_cost_usd);
        });

        const getUsagePercent = (row) => {
            if (!row.budget || row.budget <= 0) return 0;
            return (row.cycle_cost_usd / row.budget) * 100;
        };

        const getRemainingPercent = (row) => {
            if (!row.budget) return 100;
            const remain = row.cutoff_percent - getUsagePercent(row);
            return Math.max(0, remain).toFixed(2);
        };

        const getBarColor = (row) => {
            const usage = getUsagePercent(row);
            if (usage >= row.cutoff_percent) return 'bg-red-500';
            if (usage >= row.cutoff_percent * 0.85) return 'bg-yellow-500';
            return 'bg-emerald-500';
        };

        const getRemainingColor = (row) => {
            const remain = parseFloat(getRemainingPercent(row));
            if (remain <= 0) return 'text-red-400 animate-pulse';
            if (remain <= row.cutoff_percent * 0.15) return 'text-yellow-400';
            return 'text-emerald-400';
        };

        const fetchData = async () => {
            if (currentTab.value !== 'dashboard') return;
            try {
                const res = await fetch(`/api/stats?start=${startDate.value}&end=${endDate.value}`);
                const json = await res.json();
                apiData.value = json.details || [];
                const accSet = new Set(apiData.value.map(d => d.account));
                availableAccounts.value = Array.from(accSet).map(a => ({ account: a, label: a, value: a }));
                concurrency.value = { active: json.active_count || 0, waiting: json.waiting_count || 0, max: json.max_limit || 0 };
            } catch (e) {
                console.error("Dashboard数据抓取失败", e);
            }
        };

        const fetchSettings = async () => {
            try {
                const res = await fetch('/api/admin/settings');
                settings.value = await res.json();
            } catch (e) { console.error(e) }
        };

        const fetchNodes = async () => {
            try {
                const res = await fetch('/api/admin/nodes');
                nodes.value = await res.json() || [];
            } catch (e) { console.error(e) }
        };

        const saveSettings = async () => {
            if (settings.value.breaker.failure_threshold < 0 || 
                settings.value.breaker.failure_window_seconds < 0 || 
                settings.value.breaker.initial_cooldown_seconds < 0 || 
                settings.value.breaker.max_cooldown_seconds < 0) {
                showToast('各项设置的值不能为负数', 'error');
                return;
            }
            
            try {
                const payload = {
                    listen_addr: settings.value.listen_addr,
                    initial_cooldown_seconds: settings.value.breaker.initial_cooldown_seconds,
                    max_cooldown_seconds: settings.value.breaker.max_cooldown_seconds,
                    failure_threshold: settings.value.breaker.failure_threshold,
                    failure_window_seconds: settings.value.breaker.failure_window_seconds
                };
                const res = await fetch('/api/admin/settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (res.ok) {
                    showToast('系统设置已保存并热加载生效');
                } else {
                    showToast('保存失败', 'error');
                }
            } catch(e) {
                showToast('网络错误', 'error');
            }
        };

        const resetSettings = () => {
            if(!confirm('确定要恢复系统默认设置吗？')) return;
            settings.value = {
                listen_addr: '127.0.0.1:28888',
                breaker: {
                    initial_cooldown_seconds: 60,
                    max_cooldown_seconds: 3600,
                    failure_threshold: 3,
                    failure_window_seconds: 120
                }
            };
        };

        const openNodeModal = (node = null) => {
            if (node) {
                nodeForm.value = { ...node, key: '' }; // Don't show real key
                nodeModal.value = { show: true, isEdit: true };
            } else {
                nodeForm.value = {
                    id: 0, platform: 'openai', name: '', key: '', project_id: '', location: 'global', base_url: '',
                    priority: 0, cutoff_percent: 95.0, budget: 0.0, billing_start_date: '2000-01-01', is_enabled: true
                };
                nodeModal.value = { show: true, isEdit: false };
            }
        };

        const saveNode = async () => {
            if (!nodeForm.value.name || (!nodeModal.value.isEdit && !nodeForm.value.key)) {
                showToast('节点名称和Key不能为空', 'error');
                return;
            }
            if (nodeForm.value.platform === 'vertex' && !nodeForm.value.project_id) {
                showToast('GCP Project ID 不能为空', 'error');
                return;
            }
            if (nodeForm.value.priority < 0 || nodeForm.value.budget < 0 || nodeForm.value.cutoff_percent < 0) {
                showToast('优先级、额度等数字不能为负数', 'error');
                return;
            }
            if (nodeForm.value.cutoff_percent > 100) {
                showToast('阻断水位线不能超过100', 'error');
                return;
            }
            
            const dateRegex = /^\d{4}-\d{2}-\d{2}$/;
            if (!dateRegex.test(nodeForm.value.billing_start_date)) {
                showToast('日期格式必须为 YYYY-MM-DD', 'error');
                return;
            }

            try {
                const method = nodeModal.value.isEdit ? 'PUT' : 'POST';
                const res = await fetch('/api/admin/nodes', {
                    method,
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(nodeForm.value)
                });
                if (res.ok) {
                    showToast(nodeModal.value.isEdit ? '节点已更新' : '节点已添加');
                    nodeModal.value.show = false;
                    fetchNodes();
                } else {
                    const err = await res.text();
                    showToast('保存失败: ' + err, 'error');
                }
            } catch(e) {
                showToast('网络错误', 'error');
            }
        };

        const deleteNode = async (id) => {
            if(!confirm('确定要删除这个节点吗？此操作不可恢复。')) return;
            try {
                const res = await fetch(`/api/admin/nodes?id=${id}`, { method: 'DELETE' });
                if (res.ok) {
                    showToast('节点已删除');
                    fetchNodes();
                } else {
                    showToast('删除失败', 'error');
                }
            } catch(e) {
                showToast('网络错误', 'error');
            }
        };

        const fetchLogs = async () => {
            if (currentTab.value !== 'logs') return;
            try {
                const res = await fetch('/api/admin/logs');
                const rawText = await res.text();
                let lines = rawText.split('\n');
                if (logLevelFilter.value !== 'all') {
                    const levelStr = `level=${logLevelFilter.value.toUpperCase()}`;
                    lines = lines.filter(line => line.includes(levelStr) || line.trim() === '');
                }
                logsText.value = lines.join('\n');
                
                if (isAutoScroll.value) {
                    Vue.nextTick(() => {
                        const container = document.getElementById('logContainer');
                        if (container) container.scrollTop = container.scrollHeight;
                    });
                }
            } catch (e) {
                console.error("Fetch logs failed", e);
            }
        };

        const updateDateRange = (start, end, presetName) => {
            startDate.value = start; endDate.value = end; activePreset.value = presetName;
            if (fpInstance) fpInstance.setDate([start, end]);
            fetchData();
        };

        const formatDate = (date) => {
            const d = new Date(date);
            return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
        };

        const setPreset = (preset) => {
            const today = new Date(); let start = new Date();
            if (preset === 'today') start = today;
            else if (preset === 'week') start.setDate(today.getDate() - 6);
            else if (preset === 'month') start = new Date(today.getFullYear(), today.getMonth(), 1);
            updateDateRange(formatDate(start), formatDate(today), preset);
        };

        const aggregatedData = computed(() => {
            let source = apiData.value;
            if (selectedAccount.value !== 'all') {
                source = source.filter(d => d.account === selectedAccount.value);
            }
            let tCost = 0, tPrompt = 0, tComp = 0, tErr = 0, tSucc = 0;
            source.forEach(row => {
                tCost += row.period_cost_usd; tPrompt += row.prompt_tokens;
                tComp += row.completion_tokens; tErr += row.error_count; tSucc += row.success_count;
            });
            let rate = 0;
            if (tSucc + tErr > 0) rate = ((tSucc / (tSucc + tErr)) * 100).toFixed(2);
            return { totalCost: tCost, totalPrompt: tPrompt, totalCompletion: tComp, totalError: tErr, totalSuccess: tSucc, successRate: rate };
        });

        Vue.watch(currentTab, (newTab) => {
            if (newTab === 'settings') fetchSettings();
            if (newTab === 'nodes') fetchNodes();
            if (newTab === 'dashboard') fetchData();
            if (newTab === 'logs') fetchLogs();
        });

        onMounted(() => {
            fpInstance = flatpickr("#datePicker", {
                mode: "range", dateFormat: "Y-m-d", locale: "zh",
                onChange: (selectedDates) => {
                    if (selectedDates.length === 2) {
                        activePreset.value = 'custom';
                        startDate.value = formatDate(selectedDates[0]);
                        endDate.value = formatDate(selectedDates[1]);
                        fetchData();
                    }
                }
            });
            setPreset('today');
            setInterval(() => {
                fetchData();
                fetchLogs();
            }, 3000);
        });

        return {
            currentTab, apiData, availableAccounts, selectedAccount, selectedAccountLabel, activePreset, groupedApiData, singleAccountDetails,
            setPreset, aggregatedData, formatNum, formatToken, formatShortDate, successRateColor, concurrency,
            getUsagePercent, getRemainingPercent, getBarColor, getRemainingColor,
            settings, nodes, fetchSettings, fetchNodes, saveSettings, resetSettings,
            nodeModal, nodeForm, openNodeModal, saveNode, deleteNode, toast,
            logsText, isAutoScroll, logLevelFilter, fetchLogs
        };
    }
}).mount('#app');
