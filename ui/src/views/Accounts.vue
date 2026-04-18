<template>
  <div class="max-w-6xl mx-auto space-y-6">
    <div class="flex justify-between items-center mb-8">
      <div>
        <h2 class="text-2xl font-bold text-slate-800">{{ $t('nav.accounts') }}</h2>
        <p class="text-slate-500 text-sm mt-1">Manage physical API keys and monitor Sentinel health checks.</p>
      </div>
      <button 
        @click="showCreateModal = true"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center"
      >
        <svg class="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Account
      </button>
    </div>

    <div class="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="bg-slate-50 text-slate-500 border-b border-slate-100">
            <tr>
              <th class="px-6 py-3 font-medium">Provider</th>
              <th class="px-6 py-3 font-medium">Physical Key</th>
              <th class="px-6 py-3 font-medium">Priority</th>
              <th class="px-6 py-3 font-medium">{{ $t('common.status') }}</th>
              <th class="px-6 py-3 font-medium text-right">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="account in accounts" :key="account.id" class="hover:bg-slate-50/50 transition-colors">
              <td class="px-6 py-4 font-medium text-slate-700">{{ account.provider_name }}</td>
              <td class="px-6 py-4 font-mono text-xs text-slate-500">
                {{ maskKey(account.api_key) }}
              </td>
              <td class="px-6 py-4 text-slate-600">Level {{ account.priority }}</td>
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
                <button @click="editAccount(account)" class="text-slate-400 hover:text-blue-600 text-sm font-medium transition-colors mr-3">
                  {{ $t('common.edit') }}
                </button>
                <button @click="confirmDelete(account.id)" class="text-red-500 hover:text-red-700 text-sm font-medium transition-colors">
                  {{ $t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { AccountsAPI } from '../api'

const accounts = ref([])
const showCreateModal = ref(false)

const maskKey = (key) => {
  if (!key) return ''
  return `sk-...${key.substring(key.length - 4)}`
}

const confirmDelete = async (id) => {
  if (confirm('确认删除此账号？')) {
    await AccountsAPI.delete(id)
    await loadAccounts()
  }
}

const editAccount = (account) => {
  alert(`编辑账号: ${account.provider_name}`)
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