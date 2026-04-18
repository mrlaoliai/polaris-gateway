<template>
  <div class="max-w-7xl mx-auto space-y-6">

    <!-- 页头 -->
    <div class="flex justify-between items-center">
      <div>
        <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">{{ $t('nav.providers') }}</h2>
        <p class="text-slate-500 text-sm mt-1">管理 AI 厂商与旗下模型版本信息</p>
      </div>
      <button @click="openAddProvider" class="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center gap-2">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
        </svg>
        添加厂商
      </button>
    </div>

    <!-- 主体：左厂商列表 + 右模型列表 -->
    <div class="flex gap-6 min-h-[600px]">

      <!-- ── 左侧：厂商列表 ───────────────────────────── -->
      <div class="w-80 shrink-0 flex flex-col gap-3">
        <div v-if="providers.length === 0" class="flex-1 flex items-center justify-center text-slate-400 text-sm border border-dashed border-slate-200 dark:border-slate-700 rounded-xl">
          暂无厂商数据
        </div>
        <div
          v-for="p in providers" :key="p.id"
          @click="selectProvider(p)"
          :class="[
            'p-4 rounded-xl border cursor-pointer transition-all',
            selectedProvider?.id === p.id
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/30 shadow-sm'
              : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 hover:border-indigo-300 hover:shadow-sm'
          ]"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0 mr-2">
              <div class="font-semibold text-slate-800 dark:text-slate-100 truncate">{{ p.name }}</div>
              <div class="text-xs text-slate-400 mt-0.5 truncate">{{ p.protocol_type }}</div>
              <div class="text-xs text-slate-400 mt-0.5 truncate" :title="p.base_url">{{ p.base_url }}</div>
            </div>
            <span class="shrink-0 text-xs bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-0.5 rounded-full">
              {{ p.model_count }} 模型
            </span>
          </div>
          <div class="flex gap-2 mt-3">
            <button
              @click.stop="openEditProvider(p)"
              class="flex-1 text-xs text-indigo-600 hover:text-indigo-800 border border-indigo-200 hover:border-indigo-400 rounded-md py-1 transition-colors"
            >编辑</button>
            <button
              @click.stop="confirmDeleteProvider(p)"
              class="flex-1 text-xs text-red-500 hover:text-red-700 border border-red-200 hover:border-red-400 rounded-md py-1 transition-colors"
            >删除</button>
          </div>
        </div>
      </div>

      <!-- ── 右侧：模型列表 ───────────────────────────── -->
      <div class="flex-1 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden flex flex-col">
        <!-- 右侧顶部工具栏 -->
        <div class="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
          <div>
            <span class="font-semibold text-slate-800 dark:text-slate-100" v-if="selectedProvider">
              {{ selectedProvider.name }} — 模型列表
            </span>
            <span class="text-slate-400 text-sm" v-else>← 点击左侧厂商查看模型</span>
          </div>
          <button
            v-if="selectedProvider"
            @click="openAddModel"
            class="bg-emerald-600 hover:bg-emerald-700 text-white px-3 py-1.5 rounded-lg text-xs font-medium transition-colors flex items-center gap-1"
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            添加模型
          </button>
        </div>

        <!-- 模型表格 -->
        <div class="overflow-y-auto flex-1">
          <div v-if="!selectedProvider" class="flex-1 flex items-center justify-center h-full text-slate-300 py-20 text-sm">
            请先在左侧选择一个厂商
          </div>
          <div v-else-if="models.length === 0" class="flex items-center justify-center h-full text-slate-400 py-20 text-sm">
            该厂商暂无模型，点击右上角添加
          </div>
          <table v-else class="w-full text-sm text-left">
            <thead class="bg-slate-50 dark:bg-slate-800/50 text-slate-500 text-xs uppercase border-b border-slate-100 dark:border-slate-800 sticky top-0">
              <tr>
                <th class="px-6 py-3 font-medium">模型名称</th>
                <th class="px-6 py-3 font-medium">工具格式</th>
                <th class="px-6 py-3 font-medium text-center">思维链</th>
                <th class="px-6 py-3 font-medium text-center">视觉</th>
                <th class="px-6 py-3 font-medium">DSL 规则</th>
                <th class="px-6 py-3 font-medium text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
              <tr v-for="m in models" :key="m.id" class="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
                <td class="px-6 py-3 font-mono text-xs text-slate-700 dark:text-slate-200">{{ m.model_name }}</td>
                <td class="px-6 py-3 text-slate-500">{{ m.tool_format || '-' }}</td>
                <td class="px-6 py-3 text-center">
                  <span v-if="m.supports_thinking" class="text-emerald-500">🧠</span>
                  <span v-else class="text-slate-300">—</span>
                </td>
                <td class="px-6 py-3 text-center">
                  <span v-if="m.supports_vision" class="text-blue-500">👁️</span>
                  <span v-else class="text-slate-300">—</span>
                </td>
                <td class="px-6 py-3">
                  <span v-if="m.dsl_rules" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-50 text-amber-700 border border-amber-100">
                    CEL-go
                  </span>
                  <span v-else class="text-slate-300 text-xs">-</span>
                </td>
                <td class="px-6 py-3 text-right space-x-3">
                  <button @click="openEditModel(m)" class="text-indigo-500 hover:text-indigo-700 text-xs font-medium transition-colors">编辑</button>
                  <button @click="confirmDeleteModel(m)" class="text-red-500 hover:text-red-700 text-xs font-medium transition-colors">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ══ 厂商弹窗（新增 / 编辑）══ -->
    <div v-if="showProviderModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-md p-6 space-y-4">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">
          {{ providerForm.id ? '编辑厂商' : '添加厂商' }}
        </h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">厂商名称 <span class="text-red-500">*</span></label>
            <input v-model="providerForm.name" type="text" placeholder="如: Anthropic" class="p-input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">协议类型 <span class="text-red-500">*</span></label>
            <select v-model="providerForm.protocol_type" class="p-input">
              <option value="">-- 选择协议 --</option>
              <option value="anthropic">anthropic</option>
              <option value="google">google (AI Studio)</option>
              <option value="vertex">vertex (Vertex AI)</option>
              <option value="openai">openai (兼容)</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Base URL <span class="text-red-500">*</span></label>
            <input v-model="providerForm.base_url" type="text" placeholder="https://api.example.com/v1/..." class="p-input font-mono text-xs" />
          </div>
        </div>
        <p v-if="modalError" class="text-red-500 text-sm">{{ modalError }}</p>
        <div class="flex justify-end gap-3 pt-2">
          <button @click="closeProviderModal" class="px-4 py-2 text-sm text-slate-500 hover:text-slate-800 transition-colors">取消</button>
          <button @click="submitProvider" :disabled="submitting" class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-md text-sm font-medium transition-colors">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ══ 模型弹窗（新增 / 编辑）══ -->
    <div v-if="showModelModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-lg p-6 space-y-4">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">
          {{ modelForm.id ? '编辑模型' : `添加模型 — ${selectedProvider?.name}` }}
        </h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">模型名称 <span class="text-red-500">*</span></label>
            <input v-model="modelForm.model_name" type="text" placeholder="如: claude-3-5-sonnet-20241022" class="p-input font-mono text-xs" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">工具调用格式</label>
            <select v-model="modelForm.tool_format" class="p-input">
              <option value="">无</option>
              <option value="openai">openai</option>
              <option value="anthropic">anthropic</option>
              <option value="google">google</option>
            </select>
          </div>
          <div class="flex gap-6">
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="modelForm.supports_thinking" class="w-4 h-4 rounded accent-indigo-600" />
              <span class="text-sm text-slate-700 dark:text-slate-300">支持思维链 🧠</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="modelForm.supports_vision" class="w-4 h-4 rounded accent-indigo-600" />
              <span class="text-sm text-slate-700 dark:text-slate-300">支持视觉 👁️</span>
            </label>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              DSL 规则 <span class="text-slate-400 font-normal text-xs">(CEL 表达式，可选)</span>
            </label>
            <textarea
              v-model="modelForm.dsl_rules"
              rows="3"
              placeholder='例: msg.model == "claude-3-opus" ? "deepseek-v3" : ""'
              class="p-input font-mono text-xs resize-none"
            ></textarea>
          </div>
        </div>
        <p v-if="modalError" class="text-red-500 text-sm">{{ modalError }}</p>
        <div class="flex justify-end gap-3 pt-2">
          <button @click="closeModelModal" class="px-4 py-2 text-sm text-slate-500 hover:text-slate-800 transition-colors">取消</button>
          <button @click="submitModel" :disabled="submitting" class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white rounded-md text-sm font-medium transition-colors">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

// ── 状态 ─────────────────────────────────────────────────────
const providers = ref([])
const models = ref([])
const selectedProvider = ref(null)

const showProviderModal = ref(false)
const showModelModal = ref(false)
const submitting = ref(false)
const modalError = ref('')

const providerForm = ref({ id: null, name: '', protocol_type: '', base_url: '' })
const modelForm = ref({ id: null, model_name: '', tool_format: '', supports_thinking: false, supports_vision: false, dsl_rules: '' })

// ── 厂商操作 ──────────────────────────────────────────────────
const loadProviders = async () => {
  try {
    const res = await axios.get('/api/v1/providers')
    providers.value = res.data
    // 如果当前有选中厂商，刷新其 model_count
    if (selectedProvider.value) {
      const updated = res.data.find(p => p.id === selectedProvider.value.id)
      if (updated) selectedProvider.value = updated
    }
  } catch (e) {
    console.error('加载厂商列表失败', e)
  }
}

const selectProvider = async (p) => {
  selectedProvider.value = p
  await loadModels(p.id)
}

const openAddProvider = () => {
  providerForm.value = { id: null, name: '', protocol_type: '', base_url: '' }
  modalError.value = ''
  showProviderModal.value = true
}

const openEditProvider = (p) => {
  providerForm.value = { id: p.id, name: p.name, protocol_type: p.protocol_type, base_url: p.base_url }
  modalError.value = ''
  showProviderModal.value = true
}

const closeProviderModal = () => {
  showProviderModal.value = false
  modalError.value = ''
}

const submitProvider = async () => {
  modalError.value = ''
  const f = providerForm.value
  if (!f.name || !f.protocol_type || !f.base_url) {
    modalError.value = '请填写所有必填字段'
    return
  }
  submitting.value = true
  try {
    if (f.id) {
      await axios.put(`/api/v1/providers/${f.id}`, { name: f.name, protocol_type: f.protocol_type, base_url: f.base_url })
    } else {
      await axios.post('/api/v1/providers', { name: f.name, protocol_type: f.protocol_type, base_url: f.base_url })
    }
    closeProviderModal()
    await loadProviders()
  } catch (e) {
    modalError.value = e.response?.data?.error || '保存失败'
  } finally {
    submitting.value = false
  }
}

const confirmDeleteProvider = async (p) => {
  if (!confirm(`确认删除厂商【${p.name}】？\n该厂商下所有模型规格也将被一并删除。`)) return
  try {
    await axios.delete(`/api/v1/providers/${p.id}`)
    if (selectedProvider.value?.id === p.id) {
      selectedProvider.value = null
      models.value = []
    }
    await loadProviders()
  } catch (e) {
    alert('删除失败: ' + (e.response?.data?.error || e.message))
  }
}

// ── 模型操作 ──────────────────────────────────────────────────
const loadModels = async (providerId) => {
  try {
    const res = await axios.get(`/api/v1/model-specs?provider_id=${providerId}`)
    models.value = res.data
  } catch (e) {
    models.value = []
  }
}

const openAddModel = () => {
  modelForm.value = { id: null, model_name: '', tool_format: '', supports_thinking: false, supports_vision: false, dsl_rules: '' }
  modalError.value = ''
  showModelModal.value = true
}

const openEditModel = (m) => {
  modelForm.value = {
    id: m.id,
    model_name: m.model_name,
    tool_format: m.tool_format || '',
    supports_thinking: m.supports_thinking,
    supports_vision: m.supports_vision,
    dsl_rules: m.dsl_rules || ''
  }
  modalError.value = ''
  showModelModal.value = true
}

const closeModelModal = () => {
  showModelModal.value = false
  modalError.value = ''
}

const submitModel = async () => {
  modalError.value = ''
  const f = modelForm.value
  if (!f.model_name.trim()) {
    modalError.value = '模型名称不能为空'
    return
  }
  submitting.value = true
  const payload = {
    model_name: f.model_name.trim(),
    tool_format: f.tool_format,
    supports_thinking: f.supports_thinking,
    supports_vision: f.supports_vision,
    dsl_rules: f.dsl_rules
  }
  try {
    if (f.id) {
      await axios.put(`/api/v1/model-specs/${f.id}`, payload)
    } else {
      await axios.post('/api/v1/model-specs', { ...payload, provider_id: selectedProvider.value.id })
    }
    closeModelModal()
    await loadModels(selectedProvider.value.id)
    await loadProviders() // 刷新 model_count
  } catch (e) {
    modalError.value = e.response?.data?.error || '保存失败'
  } finally {
    submitting.value = false
  }
}

const confirmDeleteModel = async (m) => {
  if (!confirm(`确认删除模型【${m.model_name}】？`)) return
  try {
    await axios.delete(`/api/v1/model-specs/${m.id}`)
    await loadModels(selectedProvider.value.id)
    await loadProviders()
  } catch (e) {
    alert('删除失败: ' + (e.response?.data?.error || e.message))
  }
}

onMounted(loadProviders)
</script>
