<template>
  <div class="max-w-6xl mx-auto space-y-6">
    <div class="flex justify-between items-center mb-8">
      <div>
        <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">{{ $t('nav.accounts') }}</h2>
        <p class="text-slate-500 text-sm mt-1">Manage physical API keys and monitor Sentinel health checks.</p>
      </div>
      <button
        @click="openCreateModal"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center"
      >
        <svg class="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Account
      </button>
    </div>

    <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="bg-slate-50 dark:bg-slate-800/50 text-slate-500 border-b border-slate-100 dark:border-slate-800">
            <tr>
              <th class="px-6 py-3 font-medium">Provider</th>
              <th class="px-6 py-3 font-medium">Physical Key</th>
              <th class="px-6 py-3 font-medium">Priority</th>
              <th class="px-6 py-3 font-medium">{{ $t('common.status') }}</th>
              <th class="px-6 py-3 font-medium text-right">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
            <tr v-if="accounts.length === 0">
              <td colspan="5" class="px-6 py-12 text-center text-slate-400 text-sm">暂无账号，点击右上角添加</td>
            </tr>
            <tr v-for="account in accounts" :key="account.id" class="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
              <td class="px-6 py-4 font-medium text-slate-700 dark:text-slate-200">{{ account.provider_name }}</td>
              <td class="px-6 py-4 font-mono text-xs text-slate-500">
                {{ maskKey(account.api_key) }}
              </td>
              <td class="px-6 py-4 text-slate-600 dark:text-slate-400">Level {{ account.priority }}</td>
              <td class="px-6 py-4">
                <span v-if="account.status === 'active'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-50 text-emerald-600 border border-emerald-100">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1.5"></span>
                  Active
                </span>
                <span v-else class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-50 text-red-600 border border-red-100">
                  <span class="w-1.5 h-1.5 rounded-full bg-red-500 mr-1.5"></span>
                  Sentinel Error
                </span>
              </td>
              <td class="px-6 py-4 text-right">
                <button @click="confirmDelete(account.id)" class="text-red-500 hover:text-red-700 text-sm font-medium transition-colors">
                  {{ $t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 添加账号弹窗 -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-lg p-6 space-y-5">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">添加物理账号</h3>

        <div class="space-y-4">
          <!-- Provider 选择 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Provider <span class="text-red-500">*</span>
            </label>
            <select
              v-model.number="form.providerId"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm"
            >
              <option value="" disabled>-- 选择 Provider --</option>
              <option v-for="p in providers" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.protocol_type }})
              </option>
            </select>
            <p v-if="providers.length === 0" class="text-xs text-amber-500 mt-1">
              未找到 Provider，请先在数据库中添加 providers 记录
            </p>
          </div>

          <!-- API Key -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              API Key <span class="text-red-500">*</span>
            </label>
            <input
              v-model="form.apiKey"
              type="password"
              placeholder="sk-..."
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm font-mono"
            />
          </div>

          <!-- 优先级 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Priority <span class="text-slate-400 font-normal">(数字越大优先级越高，默认 10)</span>
            </label>
            <input
              v-model.number="form.priority"
              type="number"
              min="1"
              max="100"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm"
            />
          </div>
        </div>

        <!-- 错误提示 -->
        <p v-if="formError" class="text-red-500 text-sm">{{ formError }}</p>

        <div class="flex justify-end space-x-3 pt-2">
          <button @click="closeModal" class="px-4 py-2 text-sm text-slate-600 dark:text-slate-400 hover:text-slate-800 transition-colors">
            {{ $t('common.cancel') }}
          </button>
          <button
            @click="handleCreate"
            :disabled="submitting"
            class="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-md text-sm font-medium transition-colors"
          >
            {{ submitting ? '提交中...' : $t('common.confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { AccountsAPI } from '../api'
import axios from 'axios'

const accounts = ref([])
const providers = ref([])
const showCreateModal = ref(false)
const submitting = ref(false)
const formError = ref('')

const form = ref({
  providerId: '',
  apiKey: '',
  priority: 10
})

const maskKey = (key) => {
  if (!key) return ''
  return `sk-...${key.substring(key.length - 4)}`
}

const openCreateModal = async () => {
  // 打开弹窗时先拉取 provider 列表
  try {
    const res = await axios.get('/api/v1/providers')
    providers.value = res.data
  } catch (e) {
    providers.value = []
  }
  showCreateModal.value = true
}

const closeModal = () => {
  showCreateModal.value = false
  formError.value = ''
  form.value = { providerId: '', apiKey: '', priority: 10 }
}

const confirmDelete = async (id) => {
  if (confirm('确认删除此账号？')) {
    await AccountsAPI.delete(id)
    await loadAccounts()
  }
}

const handleCreate = async () => {
  formError.value = ''
  if (!form.value.providerId) {
    formError.value = '请选择 Provider'
    return
  }
  if (!form.value.apiKey.trim()) {
    formError.value = 'API Key 不能为空'
    return
  }
  submitting.value = true
  try {
    await AccountsAPI.create({
      provider_id: form.value.providerId,
      api_key: form.value.apiKey.trim(),
      priority: form.value.priority || 10
    })
    closeModal()
    await loadAccounts()
  } catch (e) {
    formError.value = '创建失败，请检查参数'
    console.error(e)
  } finally {
    submitting.value = false
  }
}

const loadAccounts = async () => {
  try {
    accounts.value = await AccountsAPI.list()
  } catch (e) {
    console.error('获取账号列表失败', e)
  }
}

onMounted(loadAccounts)
</script>