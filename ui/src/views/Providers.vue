<template>
  <div class="max-w-7xl mx-auto space-y-6">

    <!-- 页头 -->
    <div class="flex justify-between items-center">
      <div>
        <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">{{ $t('nav.providers') }}</h2>
        <p class="text-slate-500 text-sm mt-1">管理 AI 厂商系统配置与旗下模型版本信息</p>
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
      <div class="w-80 shrink-0 flex flex-col gap-3 overflow-y-auto pb-4">
        <div v-if="providers.length === 0" class="flex-1 flex items-center justify-center text-slate-400 text-sm border border-dashed border-slate-200 dark:border-slate-700 rounded-xl py-20">
          暂无厂商配置
        </div>
        <div
          v-for="p in providers" :key="p.id"
          @click="selectProvider(p)"
          :class="[
            'p-4 rounded-xl border cursor-pointer transition-all select-none',
            selectedProvider?.id === p.id
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/30 shadow-sm'
              : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 hover:border-indigo-300 hover:shadow-sm'
          ]"
        >
          <!-- 厂商标识 + 协议徽章 -->
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="font-semibold text-slate-800 dark:text-slate-100 text-sm truncate">{{ p.name }}</div>
              <div class="text-xs text-slate-400 mt-0.5 font-mono truncate">{{ p.id }}</div>
            </div>
            <span :class="protocolBadgeClass(p.protocol)" class="shrink-0 text-xs px-2 py-0.5 rounded-full font-medium">
              {{ p.protocol }}
            </span>
          </div>

          <!-- URL 模板 -->
          <div class="mt-2 text-xs text-slate-400 font-mono truncate" :title="p.url_template">
            {{ p.url_template }}
          </div>

          <!-- 认证类型 + 模型数 -->
          <div class="flex items-center justify-between mt-2">
            <span class="text-xs text-slate-500">
              🔑 {{ p.auth_type }}
              · ⏱ {{ p.read_timeout }}s
            </span>
            <span class="text-xs bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 px-2 py-0.5 rounded-full">
              {{ p.model_count }} 模型
            </span>
          </div>

          <div class="flex gap-2 mt-3">
            <button @click.stop="openEditProvider(p)" class="flex-1 text-xs text-indigo-600 hover:text-indigo-800 border border-indigo-200 hover:border-indigo-400 rounded-md py-1 transition-colors">编辑</button>
            <button @click.stop="confirmDeleteProvider(p)" class="flex-1 text-xs text-red-500 hover:text-red-700 border border-red-200 hover:border-red-400 rounded-md py-1 transition-colors">删除</button>
          </div>
        </div>
      </div>

      <!-- ── 右侧：模型列表 ───────────────────────────── -->
      <div class="flex-1 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden flex flex-col">
        <div class="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
          <div>
            <span class="font-semibold text-slate-800 dark:text-slate-100" v-if="selectedProvider">
              {{ selectedProvider.name }} — 模型列表
            </span>
            <span class="text-slate-400 text-sm" v-else>← 点击左侧厂商查看模型</span>
          </div>
          <button v-if="selectedProvider" @click="openAddModel"
            class="bg-emerald-600 hover:bg-emerald-700 text-white px-3 py-1.5 rounded-lg text-xs font-medium transition-colors flex items-center gap-1">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            添加模型
          </button>
        </div>

        <div class="overflow-auto flex-1">
          <div v-if="!selectedProvider" class="flex items-center justify-center h-full text-slate-300 py-20 text-sm">
            请先在左侧选择一个厂商
          </div>
          <div v-else-if="models.length === 0" class="flex items-center justify-center h-full text-slate-400 py-20 text-sm">
            该厂商暂无模型，点击右上角添加
          </div>
          <table v-else class="w-full text-sm text-left">
            <thead class="bg-slate-50 dark:bg-slate-800/50 text-slate-500 text-xs border-b border-slate-100 dark:border-slate-800 sticky top-0">
              <tr>
                <th class="px-4 py-3 font-medium">物理 Model ID</th>
                <th class="px-4 py-3 font-medium">展示名</th>
                <th class="px-4 py-3 font-medium text-right">上下文</th>
                <th class="px-4 py-3 font-medium text-center">能力</th>
                <th class="px-4 py-3 font-medium">DSL</th>
                <th class="px-4 py-3 font-medium text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
              <tr v-for="m in models" :key="m.id" class="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
                <td class="px-4 py-3 font-mono text-xs text-slate-600 dark:text-slate-300">{{ m.model_id }}</td>
                <td class="px-4 py-3 text-slate-700 dark:text-slate-200 text-xs">{{ m.model_name }}</td>
                <td class="px-4 py-3 text-right text-xs text-slate-500">{{ formatCtx(m.max_context) }}</td>
                <td class="px-4 py-3 text-center">
                  <div class="flex items-center justify-center gap-1 flex-wrap">
                    <span v-if="m.supports_thinking" title="思维链" class="text-sm">🧠</span>
                    <span v-if="m.supports_vision"   title="视觉"   class="text-sm">👁️</span>
                    <span v-if="m.supports_tools"    title="工具调用" class="text-sm">🔧</span>
                    <span v-if="m.supports_json"     title="JSON Mode" class="text-sm">📋</span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span v-if="m.dsl_rules" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-amber-50 text-amber-700 border border-amber-100">DSL</span>
                  <span v-else class="text-slate-300 text-xs">-</span>
                </td>
                <td class="px-4 py-3 text-right space-x-3">
                  <button @click="openEditModel(m)" class="text-indigo-500 hover:text-indigo-700 text-xs font-medium transition-colors">编辑</button>
                  <button @click="confirmDeleteModel(m)" class="text-red-500 hover:text-red-700 text-xs font-medium transition-colors">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ══ 厂商弹窗（新增 / 编辑）══ ─────────────────────── -->
    <div v-if="showProviderModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">
          {{ providerForm.isEdit ? '编辑厂商' : '添加厂商' }}
        </h3>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="p-label">标识 ID <span class="text-red-500">*</span></label>
            <input v-model="providerForm.id" type="text" :disabled="providerForm.isEdit"
              placeholder="如: my-openai-proxy"
              class="p-input text-xs font-mono" :class="{'opacity-50 cursor-not-allowed': providerForm.isEdit}" />
            <p class="text-xs text-slate-400 mt-0.5">唯一标识符，创建后不可修改</p>
          </div>
          <div>
            <label class="p-label">厂商名称 <span class="text-red-500">*</span></label>
            <input v-model="providerForm.name" type="text" placeholder="如: My OpenAI Proxy" class="p-input" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="p-label">协议类型 <span class="text-red-500">*</span></label>
            <select v-model="providerForm.protocol" class="p-input">
              <option value="">-- 选择协议 --</option>
              <option value="openai">openai（兼容）</option>
              <option value="anthropic">anthropic</option>
              <option value="google-ai">google-ai（AI Studio）</option>
              <option value="vertex">vertex（Vertex AI）</option>
            </select>
          </div>
          <div>
            <label class="p-label">认证类型 <span class="text-red-500">*</span></label>
            <select v-model="providerForm.auth_type" class="p-input">
              <option value="api-key">api-key</option>
              <option value="oauth2">oauth2</option>
            </select>
          </div>
        </div>

        <div>
          <label class="p-label">URL 模板 <span class="text-red-500">*</span></label>
          <input v-model="providerForm.url_template" type="text"
            placeholder="https://api.example.com/v1/{model_id}" class="p-input text-xs font-mono" />
          <p class="text-xs text-slate-400 mt-0.5">支持 {model_id} 和 {region} 占位符</p>
        </div>

        <div>
          <label class="p-label">认证配置 (JSON)</label>
          <textarea v-model="providerForm.auth_config" rows="2"
            placeholder='{"location":"header","key_name":"Authorization","prefix":"Bearer "}'
            class="p-input text-xs font-mono resize-none"></textarea>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="p-label">连接超时 (秒)</label>
            <input v-model.number="providerForm.conn_timeout" type="number" min="1" class="p-input" />
          </div>
          <div>
            <label class="p-label">读取超时 (秒)</label>
            <input v-model.number="providerForm.read_timeout" type="number" min="1" class="p-input" />
          </div>
        </div>

        <div>
          <label class="p-label">扩展能力 (JSON，可选)</label>
          <textarea v-model="providerForm.capabilities" rows="2"
            placeholder='{"regions":["us-central1"]}'
            class="p-input text-xs font-mono resize-none"></textarea>
        </div>

        <p v-if="modalError" class="text-red-500 text-sm">{{ modalError }}</p>
        <div class="flex justify-end gap-3 pt-2">
          <button @click="closeProviderModal" class="px-4 py-2 text-sm text-slate-500 hover:text-slate-800 transition-colors">取消</button>
          <button @click="submitProvider" :disabled="submitting"
            class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-md text-sm font-medium transition-colors">
            {{ submitting ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ══ 模型弹窗（新增 / 编辑）══ ─────────────────────── -->
    <div v-if="showModelModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">
          {{ modelForm.id ? '编辑模型' : `添加模型 — ${selectedProvider?.name}` }}
        </h3>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="p-label">物理 Model ID <span class="text-red-500">*</span></label>
            <input v-model="modelForm.model_id" type="text" placeholder="如: gpt-5.4-omni" class="p-input text-xs font-mono" />
          </div>
          <div>
            <label class="p-label">展示名 <span class="text-red-500">*</span></label>
            <input v-model="modelForm.model_name" type="text" placeholder="如: GPT-5.4 Omni" class="p-input" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="p-label">工具调用格式</label>
            <select v-model="modelForm.tool_format" class="p-input">
              <option value="">无</option>
              <option value="openai">openai</option>
              <option value="anthropic">anthropic</option>
              <option value="google">google</option>
            </select>
          </div>
          <div>
            <label class="p-label">最大上下文 (tokens)</label>
            <input v-model.number="modelForm.max_context" type="number" placeholder="如: 1000000" class="p-input" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-x-6 gap-y-2">
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="modelForm.supports_thinking" class="w-4 h-4 rounded accent-indigo-600" />
            <span class="text-sm text-slate-700 dark:text-slate-300">支持思维链 🧠</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="modelForm.supports_vision" class="w-4 h-4 rounded accent-indigo-600" />
            <span class="text-sm text-slate-700 dark:text-slate-300">支持视觉 👁️</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="modelForm.supports_tools" class="w-4 h-4 rounded accent-indigo-600" />
            <span class="text-sm text-slate-700 dark:text-slate-300">支持 Function Call 🔧</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="modelForm.supports_json" class="w-4 h-4 rounded accent-indigo-600" />
            <span class="text-sm text-slate-700 dark:text-slate-300">支持 JSON Mode 📋</span>
          </label>
        </div>

        <div>
          <label class="p-label">DSL 规则 <span class="text-slate-400 font-normal text-xs">(可选，内置响应清洗逻辑)</span></label>
          <textarea v-model="modelForm.dsl_rules" rows="3"
            placeholder="res.content = res.content.trim()"
            class="p-input text-xs font-mono resize-none"></textarea>
        </div>

        <div>
          <label class="p-label">扩展参数 (JSON，可选)</label>
          <textarea v-model="modelForm.capabilities" rows="2"
            placeholder='{"training_cutoff":"2026-01"}'
            class="p-input text-xs font-mono resize-none"></textarea>
        </div>

        <p v-if="modalError" class="text-red-500 text-sm">{{ modalError }}</p>
        <div class="flex justify-end gap-3 pt-2">
          <button @click="closeModelModal" class="px-4 py-2 text-sm text-slate-500 hover:text-slate-800 transition-colors">取消</button>
          <button @click="submitModel" :disabled="submitting"
            class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white rounded-md text-sm font-medium transition-colors">
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

const defaultProviderForm = () => ({
  isEdit: false,
  id: '', name: '', protocol: '', url_template: '',
  auth_type: 'api-key', auth_config: '', conn_timeout: 10,
  read_timeout: 120, capabilities: ''
})

const defaultModelForm = () => ({
  id: null, model_id: '', model_name: '', tool_format: '',
  max_context: 0, supports_thinking: false, supports_vision: false,
  supports_tools: true, supports_json: true, dsl_rules: '', capabilities: ''
})

const providerForm = ref(defaultProviderForm())
const modelForm = ref(defaultModelForm())

// ── 工具函数 ──────────────────────────────────────────────────
const formatCtx = (n) => {
  if (!n) return '-'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(0)}K`
  return String(n)
}

const protocolBadgeClass = (protocol) => {
  const map = {
    'openai':     'bg-green-50 text-green-700 border border-green-100',
    'anthropic':  'bg-orange-50 text-orange-700 border border-orange-100',
    'google-ai':  'bg-blue-50 text-blue-700 border border-blue-100',
    'vertex':     'bg-purple-50 text-purple-700 border border-purple-100',
  }
  return map[protocol] || 'bg-slate-100 text-slate-600'
}

// ── 厂商操作 ──────────────────────────────────────────────────
const loadProviders = async () => {
  try {
    const res = await axios.get('/api/v1/providers')
    providers.value = res.data
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
  providerForm.value = defaultProviderForm()
  modalError.value = ''
  showProviderModal.value = true
}

const openEditProvider = (p) => {
  providerForm.value = {
    isEdit: true,
    id: p.id, name: p.name, protocol: p.protocol,
    url_template: p.url_template, auth_type: p.auth_type,
    auth_config: p.auth_config || '', conn_timeout: p.conn_timeout || 10,
    read_timeout: p.read_timeout || 120, capabilities: p.capabilities || ''
  }
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
  if (!f.id || !f.name || !f.protocol || !f.url_template || !f.auth_type) {
    modalError.value = '请填写所有必填字段'
    return
  }
  submitting.value = true
  const payload = {
    id: f.id, name: f.name, protocol: f.protocol, url_template: f.url_template,
    auth_type: f.auth_type, auth_config: f.auth_config,
    conn_timeout: f.conn_timeout, read_timeout: f.read_timeout,
    capabilities: f.capabilities
  }
  try {
    if (f.isEdit) {
      await axios.put(`/api/v1/providers/${f.id}`, payload)
    } else {
      await axios.post('/api/v1/providers', payload)
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
  if (!confirm(`确认删除厂商【${p.name}】？\n该厂商旗下所有模型也将被一并删除。`)) return
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
  modelForm.value = defaultModelForm()
  modalError.value = ''
  showModelModal.value = true
}

const openEditModel = (m) => {
  modelForm.value = { ...m }
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
  if (!f.model_id?.trim() || !f.model_name?.trim()) {
    modalError.value = '物理 Model ID 和展示名不能为空'
    return
  }
  submitting.value = true
  const payload = {
    model_id: f.model_id.trim(), model_name: f.model_name.trim(),
    tool_format: f.tool_format, max_context: f.max_context || 0,
    supports_thinking: f.supports_thinking, supports_vision: f.supports_vision,
    supports_tools: f.supports_tools, supports_json: f.supports_json,
    dsl_rules: f.dsl_rules, capabilities: f.capabilities
  }
  try {
    if (f.id) {
      await axios.put(`/api/v1/model-specs/${f.id}`, payload)
    } else {
      await axios.post('/api/v1/model-specs', { ...payload, provider_id: selectedProvider.value.id })
    }
    closeModelModal()
    await loadModels(selectedProvider.value.id)
    await loadProviders()
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

<style scoped>
.p-label { @apply block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1; }
</style>
