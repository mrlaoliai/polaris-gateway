<template>
  <div class="flex h-full gap-6">
    <!-- 左侧：用户配置的厂商列表 -->
    <div class="w-64 flex shrink-0 flex-col gap-4">
      <div class="text-xs font-semibold text-slate-400 uppercase tracking-wider px-2">Configured Providers</div>
      
      <div class="flex flex-col gap-2">
        <button v-for="p in userProviders" :key="p.id" 
                @click="selectProvider(p)"
                class="text-left px-4 py-3 rounded-lg border transition-all duration-200 flex items-center justify-between group"
                :class="selectedProvider?.id === p.id ? 'bg-white dark:bg-slate-800 border-blue-500 shadow-sm' : 'bg-slate-50 dark:bg-slate-900/50 border-transparent hover:bg-slate-100 dark:hover:bg-slate-800'">
          <div class="flex items-center gap-3 truncate">
            <span class="text-lg">{{ getProtocolEmoji(p.protocol) }}</span>
            <span class="font-medium text-slate-700 dark:text-slate-200 truncate" :title="p.name">{{ p.name }}</span>
          </div>
          <div v-if="!p.is_enabled" class="w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-600"></div>
          <div v-else class="w-2 h-2 rounded-full bg-emerald-500"></div>
        </button>

        <button @click="openAddProviderModal" 
                class="mt-2 px-4 py-3 rounded-lg bg-slate-100/50 dark:bg-slate-800/50 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 border border-dashed border-slate-300 dark:border-slate-700 transition-all font-medium flex items-center justify-center gap-2">
          <span>+ Add New Provider</span>
        </button>
      </div>
    </div>

    <!-- 右侧：选中厂商管理以及 Keys 列表 -->
    <div class="flex-1 flex flex-col min-w-0" v-if="selectedProvider">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-slate-800 dark:text-slate-100 flex items-center gap-3">
          {{ selectedProvider.name }}
          <span class="text-sm font-normal px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-500">{{ selectedProvider.protocol }}</span>
        </h2>
        <div class="flex items-center gap-2">
          <button @click="deleteProvider" class="w-9 h-9 flex items-center justify-center rounded-md border border-red-200 dark:border-red-900/30 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors" title="Delete Provider">
             🗑️
          </button>
          <button @click="openEditProviderModal" class="px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md text-sm font-medium hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors flex items-center gap-2">
            ⚙️ Edit Provider Config
          </button>
          <button @click="openAddKeyModal" class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-md text-sm font-medium transition-colors flex items-center gap-2 shadow-sm">
            + Add new key
          </button>
        </div>
      </div>

      <!-- 密钥列表 -->
      <div class="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg shadow-sm overflow-hidden flex-1 flex flex-col">
        <table class="w-full text-left text-sm" v-if="providerKeys.length > 0">
          <thead class="bg-slate-50 dark:bg-slate-800 text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-700">
            <tr>
              <th class="px-6 py-3 font-medium">API Key / Label</th>
              <th class="px-6 py-3 font-medium text-center">Weight</th>
              <th class="px-6 py-3 font-medium text-center">Status</th>
              <th class="px-6 py-3 font-medium text-center">Enabled</th>
              <th class="px-6 py-3 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
            <tr v-for="key in providerKeys" :key="key.id" class="hover:bg-slate-50/50 dark:hover:bg-slate-800/50 transition-colors">
              <td class="px-6 py-4">
                <div class="flex flex-col gap-1">
                  <div class="font-mono text-slate-700 dark:text-slate-300">
                    <span v-if="key.status === 'invalid'" class="text-red-500 mr-2" title="Invalid Key">❌</span>
                    <span v-else-if="key.status === 'cooldown'" class="text-amber-500 mr-2" title="Cooldown/Rate Limited">⏳</span>
                    <span v-else class="text-emerald-500 mr-2">✔️</span>
                    {{ key.label || key.api_key_masked }}
                  </div>
                  <div class="text-xs text-slate-400" v-if="key.label && key.credential_type === 'api-key'">{{ key.api_key_masked }}</div>
                  <div class="text-xs text-slate-400 flex items-center gap-1">
                     <span class="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 rounded">{{ key.credential_type }}</span>
                     <span v-if="key.selected_models && key.selected_models !== 'null'">{{ parseCount(key.selected_models) }} models selected</span>
                     <span v-else class="text-emerald-600/70">All models</span>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4 text-center font-medium">{{ key.weight }}</td>
              <td class="px-6 py-4 text-center">
                 <span v-if="key.status === 'active'" class="text-emerald-500 text-xs px-2 py-1 bg-emerald-50 dark:bg-emerald-900/20 rounded-full">Active</span>
                 <span v-else-if="key.status === 'cooldown'" class="text-amber-500 text-xs px-2 py-1 bg-amber-50 dark:bg-amber-900/20 rounded-full">Cooldown</span>
                 <span v-else class="text-red-500 text-xs px-2 py-1 bg-red-50 dark:bg-red-900/20 rounded-full">Invalid</span>
                 <div v-if="key.error_count > 0" class="text-[10px] text-red-400 mt-1">Err: {{ key.error_count }}</div>
              </td>
              <td class="px-6 py-4">
                <div class="flex justify-center">
                  <button @click="toggleKeyEnable(key)" class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center justify-center rounded-full transition-colors duration-200 ease-in-out focus:outline-none" :class="key.is_enabled ? 'bg-emerald-500' : 'bg-slate-200 dark:bg-slate-700'">
                    <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out" :class="key.is_enabled ? 'translate-x-2' : '-translate-x-2'"></span>
                  </button>
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <button @click="openEditKeyModal(key)" class="text-slate-400 hover:text-blue-500 px-2 transition-colors">Edit</button>
                <button @click="deleteKey(key)" class="text-slate-400 hover:text-red-500 px-2 transition-colors">Del</button>
              </td>
            </tr>
          </tbody>
        </table>
        
        <div v-else class="flex-1 flex flex-col items-center justify-center text-slate-500 p-12">
          <div class="text-4xl mb-4 opacity-50">🔑</div>
          <p>No keys configured for this provider.</p>
          <button @click="openAddKeyModal" class="mt-4 px-4 py-2 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors text-sm font-medium">Add first key</button>
        </div>
      </div>
    </div>

    <!-- 无选中时的占位 -->
    <div class="flex-1 flex flex-col items-center justify-center text-slate-400" v-else>
      <div class="text-5xl mb-4 opacity-20">⚙️</div>
      <p>Select a provider from the left sidebar or add a new one.</p>
    </div>

    <!-- 添加厂商 Modal -->
    <div v-if="showAddProviderModal" class="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div class="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-md overflow-hidden">
        <div class="px-6 py-4 border-b border-slate-100 dark:border-slate-800">
          <h3 class="font-bold text-lg">Add New Provider</h3>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">System Provider</label>
            <select v-model="addProviderForm.system_provider_id" @change="onSystemProviderSelect" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none">
              <option value="" disabled>Select from available presets...</option>
              <option v-for="sp in availableProviders" :key="sp.id" :value="sp.id">{{ sp.name }} ({{ sp.protocol }})</option>
            </select>
          </div>
          <div>
             <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Instance Name (Alias)</label>
             <input v-model="addProviderForm.name" type="text" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded focus:border-blue-500 outline-none" placeholder="e.g. My Custom OpenAI" />
          </div>
          <div>
             <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Custom Base URL (Optional)</label>
             <input v-model="addProviderForm.custom_base_url" type="text" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded focus:border-blue-500 outline-none" placeholder="Leave empty to use system default" />
          </div>
        </div>
        <div class="px-6 py-4 border-t border-slate-100 dark:border-slate-800 flex justify-end gap-3 bg-slate-50/50 dark:bg-slate-800/20">
          <button @click="showAddProviderModal = false" class="px-4 py-2 rounded text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">Cancel</button>
          <button @click="submitAddProvider" :disabled="!addProviderForm.system_provider_id" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors disabled:opacity-50">Add</button>
        </div>
      </div>
    </div>

    <!-- 编辑厂商 Modal -->
    <div v-if="showEditProviderModal" class="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div class="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-lg overflow-hidden flex flex-col max-h-[90vh]">
        <div class="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center shrink-0">
          <h3 class="font-bold text-lg">Edit Provider Config</h3>
          <div class="flex items-center gap-2">
            <span class="text-sm text-slate-500">Enabled</span>
            <input type="checkbox" v-model="editProviderForm.is_enabled" class="rounded text-blue-500 focus:ring-blue-500" />
          </div>
        </div>
        <div class="p-6 space-y-4 overflow-y-auto">
          <div>
             <label class="block text-sm font-medium mb-1">Instance Name</label>
             <input v-model="editProviderForm.name" type="text" class="w-full form-input" />
          </div>
          <div>
             <label class="block text-sm font-medium mb-1">Custom Base URL</label>
             <input v-model="editProviderForm.custom_base_url" type="text" class="w-full form-input" placeholder="0 or empty to inherit" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
               <label class="block text-sm font-medium mb-1">Conn Timeout (s)</label>
               <input v-model.number="editProviderForm.conn_timeout" type="number" class="w-full form-input" placeholder="0 = system default" />
            </div>
            <div>
               <label class="block text-sm font-medium mb-1">Read Timeout (s)</label>
               <input v-model.number="editProviderForm.read_timeout" type="number" class="w-full form-input" placeholder="0 = system default" />
            </div>
            <div>
               <label class="block text-sm font-medium mb-1">Stream Idle Timeout (s)</label>
               <input v-model.number="editProviderForm.stream_idle_timeout" type="number" class="w-full form-input" />
            </div>
            <div>
               <label class="block text-sm font-medium mb-1">Max Retries</label>
               <input v-model.number="editProviderForm.max_retries" type="number" class="w-full form-input" />
            </div>
          </div>
        </div>
        <div class="px-6 py-4 border-t border-slate-100 dark:border-slate-800 flex justify-end gap-3 shrink-0">
          <button @click="showEditProviderModal = false" class="px-4 py-2 rounded mt-btn-cancel">Cancel</button>
          <button @click="submitEditProvider" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded">Save Configuration</button>
        </div>
      </div>
    </div>

    <!-- 添加/编辑 密钥 Modal -->
    <div v-if="showKeyModal" class="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div class="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-2xl overflow-hidden flex flex-col max-h-[90vh]">
        <div class="px-6 py-4 border-b border-slate-100 dark:border-slate-800 shrink-0">
          <h3 class="font-bold text-lg">{{ isEditingKey ? 'Edit Key' : 'Add New Key' }}</h3>
        </div>
        
        <div class="p-6 space-y-4 overflow-y-auto flex-1">
          <div class="grid grid-cols-2 gap-4">
            <div>
               <label class="block text-sm font-medium mb-1">Credential Type</label>
               <select v-model="keyForm.credential_type" class="w-full form-input">
                 <option value="api-key">Standard API Key</option>
                 <option value="vertex-sa" v-if="selectedProvider?.protocol === 'vertex'">Vertex AI Service Account</option>
               </select>
            </div>
            <div>
               <label class="block text-sm font-medium mb-1">Label / Remark</label>
               <input v-model="keyForm.label" type="text" class="w-full form-input" placeholder="e.g. Sales Team Key" />
            </div>
          </div>

          <div v-if="keyForm.credential_type === 'api-key'">
             <label class="block text-sm font-medium mb-1">API Key</label>
             <input v-model="keyForm.api_key" type="password" class="w-full form-input font-mono" placeholder="sk-..." />
             <p class="text-xs text-slate-400 mt-1" v-if="isEditingKey">Leave empty to keep existing key securely.</p>
          </div>

          <template v-if="keyForm.credential_type === 'vertex-sa'">
             <div class="grid grid-cols-2 gap-4">
               <div>
                 <label class="block text-sm font-medium mb-1">Project ID</label>
                 <input v-model="keyForm.project_id" type="text" class="w-full form-input" />
               </div>
               <div>
                 <label class="block text-sm font-medium mb-1">Region</label>
                 <input v-model="keyForm.region" type="text" class="w-full form-input" placeholder="us-central1" />
               </div>
             </div>
             <div>
               <label class="block text-sm font-medium mb-1">Service Account JSON</label>
               <textarea v-model="keyForm.service_account_json" rows="3" class="w-full form-input font-mono text-xs p-2"></textarea>
               <p class="text-xs text-slate-400 mt-1" v-if="isEditingKey">Leave empty to keep existing json securely.</p>
             </div>
          </template>

          <div class="grid grid-cols-2 gap-4 mt-2">
            <div>
               <label class="block text-sm font-medium mb-1">Weight (1-100)</label>
               <input v-model.number="keyForm.weight" type="number" min="1" max="100" class="w-full form-input" />
            </div>
            <div class="flex items-center gap-2 mt-6" v-if="isEditingKey">
              <input type="checkbox" v-model="keyForm.is_enabled" id="keyEna" class="rounded text-blue-500 focus:ring-blue-500" />
              <label for="keyEna" class="text-sm font-medium">Enabled</label>
            </div>
          </div>

          <!-- 模型选择模块 -->
          <div class="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden mt-4">
             <div class="bg-slate-50 dark:bg-slate-800 px-4 py-2 border-b border-slate-200 dark:border-slate-700 flex justify-between items-center">
               <span class="font-medium text-sm">Model Authorization</span>
               <button @click="selectAllModels(false)" class="text-xs text-blue-500 hover:text-blue-600">Clear All</button>
             </div>
             <div class="p-4 bg-white dark:bg-slate-900 max-h-40 overflow-y-auto">
                <p class="text-xs text-slate-500 mb-3" v-if="keyFormSelectedModels.length === 0">
                  <span class="text-amber-500">⚠️</span> No models selected implicitly allows <strong>All models</strong> available in this provider.
                </p>
                <div class="grid grid-cols-2 gap-2">
                   <label v-for="m in currentSystemModels" :key="m.model_id" class="flex items-start gap-2 p-2 rounded hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer border border-transparent hover:border-slate-200 dark:hover:border-slate-700 transition-all">
                     <input type="checkbox" :value="m.model_id" v-model="keyFormSelectedModels" class="mt-1 rounded text-blue-500 focus:ring-blue-500" />
                     <div class="flex flex-col">
                       <span class="text-sm font-medium">{{ m.model_name }}</span>
                       <span class="text-xs text-slate-400 font-mono">{{ m.model_id }}</span>
                     </div>
                   </label>
                </div>
             </div>
          </div>

        </div>
        <div class="px-6 py-4 border-t border-slate-100 dark:border-slate-800 flex justify-end gap-3 shrink-0">
          <button @click="showKeyModal = false" class="px-4 py-2 rounded mt-btn-cancel">Cancel</button>
          <button @click="submitKey" class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded">{{ isEditingKey ? 'Save Changes' : 'Add Key' }}</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'

const userProviders = ref([])
const availableProviders = ref([])
const providerKeys = ref([])
const currentSystemModels = ref([])

const selectedProvider = ref(null)

// Modals
const showAddProviderModal = ref(false)
const showEditProviderModal = ref(false)
const showKeyModal = ref(false)

const addProviderForm = ref({ system_provider_id: '', name: '', custom_base_url: '', conn_timeout: 0, read_timeout: 0, stream_idle_timeout: 30, max_retries: 3 })
const editProviderForm = ref({})

const isEditingKey = ref(false)
const keyForm = ref({ credential_type: 'api-key', weight: 10 })
const keyFormSelectedModels = ref([]) // array of string models

const getProtocolEmoji = (proto) => {
  if (proto === 'openai') return 'O'
  if (proto === 'anthropic') return 'A'
  if (proto === 'google-ai') return 'G'
  if (proto === 'vertex') return 'V'
  return '☁️'
}

const parseCount = (arrStr) => {
  try {
    const arr = JSON.parse(arrStr)
    return arr.length
  } catch (e) { return 0 }
}

onMounted(() => {
  loadUserProviders()
})

const loadUserProviders = async () => {
  try {
    const res = await fetch('/api/v1/user-providers')
    if (res.ok) {
      userProviders.value = await res.json()
      if (!selectedProvider.value && userProviders.value.length > 0) {
        selectProvider(userProviders.value[0])
      } else if (selectedProvider.value) {
        // 更新 selectedProvider 最新状态
        const up = userProviders.value.find(p => p.id === selectedProvider.value.id)
        if (up) selectedProvider.value = up
        else selectedProvider.value = null
      }
    }
  } catch (err) { console.error('Failed to load user providers', err) }
}

const loadAvailableProviders = async () => {
  try {
    const res = await fetch('/api/v1/user-providers/available')
    if (res.ok) availableProviders.value = await res.json()
  } catch (err) {}
}

const selectProvider = async (p) => {
  selectedProvider.value = p
  loadKeys()
  loadSystemModels()
}

const loadKeys = async () => {
  if (!selectedProvider.value) return
  try {
    const res = await fetch(`/api/v1/provider-keys?user_provider_id=${selectedProvider.value.id}`)
    if (res.ok) providerKeys.value = await res.json()
  } catch (err) {}
}

const loadSystemModels = async () => {
  if (!selectedProvider.value) return
  try {
    // 根据关联的 system_provider_id 拉取物理模型
    const res = await fetch(`/api/v1/model-specs?provider_id=${selectedProvider.value.system_provider_id}`)
    if (res.ok) currentSystemModels.value = await res.json()
  } catch (err) {}
}

// ── Provider 操作 ──
const openAddProviderModal = () => {
  addProviderForm.value = { system_provider_id: '', name: '', custom_base_url: '', conn_timeout: 0, read_timeout: 0, stream_idle_timeout: 30, max_retries: 3 }
  loadAvailableProviders()
  showAddProviderModal.value = true
}

const onSystemProviderSelect = () => {
  const sp = availableProviders.value.find(x => x.id === addProviderForm.value.system_provider_id)
  if (sp && !addProviderForm.value.name) addProviderForm.value.name = sp.name
}

const submitAddProvider = async () => {
  if (!addProviderForm.value.system_provider_id) return
  try {
    const res = await fetch('/api/v1/user-providers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(addProviderForm.value)
    })
    if (res.ok) {
      showAddProviderModal.value = false
      await loadUserProviders()
      selectProvider(userProviders.value[userProviders.value.length - 1])
    } else {
      const err = await res.json()
      alert(err.error || 'Failed to add provider')
    }
  } catch (err) {}
}

const openEditProviderModal = () => {
  if (!selectedProvider.value) return
  editProviderForm.value = { ...selectedProvider.value }
  showEditProviderModal.value = true
}

const submitEditProvider = async () => {
  try {
    const res = await fetch(`/api/v1/user-providers/${editProviderForm.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(editProviderForm.value)
    })
    if (res.ok) {
      showEditProviderModal.value = false
      loadUserProviders()
    }
  } catch (err) {}
}

const deleteProvider = async () => {
  if (!selectedProvider.value) return
  if (!confirm(`Are you sure you want to delete ${selectedProvider.value.name} and ALL its keys?`)) return
  try {
    const res = await fetch(`/api/v1/user-providers/${selectedProvider.value.id}`, { method: 'DELETE' })
    if (res.ok) {
      selectedProvider.value = null
      loadUserProviders()
    }
  } catch (err) {}
}

// ── Key 操作 ──
const selectAllModels = (check) => {
  if (check && currentSystemModels.value) {
    keyFormSelectedModels.value = currentSystemModels.value.map(m => m.model_id)
  } else {
    keyFormSelectedModels.value = []
  }
}

const openAddKeyModal = () => {
  isEditingKey.value = false
  keyForm.value = { credential_type: selectedProvider.value.protocol === 'vertex' ? 'vertex-sa' : 'api-key', weight: 10, label: '', api_key: '', project_id: '', region: '', service_account_json: '' }
  keyFormSelectedModels.value = []
  showKeyModal.value = true
}

const openEditKeyModal = (key) => {
  isEditingKey.value = true
  keyForm.value = { ...key, api_key: '', service_account_json: '' } // 敏感数据不返回前台，提交空表示不修改
  if (key.selected_models && key.selected_models !== 'null' && key.selected_models !== '') {
    try {
      keyFormSelectedModels.value = JSON.parse(key.selected_models)
    } catch { keyFormSelectedModels.value = [] }
  } else {
    keyFormSelectedModels.value = []
  }
  showKeyModal.value = true
}

const toggleKeyEnable = async (key) => {
  const newVal = !key.is_enabled
  try {
    await fetch(`/api/v1/provider-keys/${key.id}/toggle`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_enabled: newVal })
    })
    loadKeys()
  } catch (e) {}
}

const submitKey = async () => {
  const body = { ...keyForm.value, user_provider_id: selectedProvider.value.id }
  body.selected_models = keyFormSelectedModels.value.length > 0 ? JSON.stringify(keyFormSelectedModels.value) : ''
  
  const method = isEditingKey.value ? 'PUT' : 'POST'
  const url = isEditingKey.value ? `/api/v1/provider-keys/${keyForm.value.id}` : '/api/v1/provider-keys'
  
  try {
    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    if (res.ok) {
      showKeyModal.value = false
      loadKeys()
    }
  } catch (err) {}
}

const deleteKey = async (key) => {
  if (!confirm(`Delete key ${key.label || 'unnamed'}?`)) return
  try {
    await fetch(`/api/v1/provider-keys/${key.id}`, { method: 'DELETE' })
    loadKeys()
  } catch (e) {}
}
</script>

<style scoped>
.form-input {
  @apply px-3 py-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded focus:border-blue-500 focus:outline-none transition-colors text-slate-900 dark:text-slate-100;
}
.mt-btn-cancel {
  @apply text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors;
}
</style>
