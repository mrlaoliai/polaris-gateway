<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-900">{{ $t('nav.gateway_keys') }}</h1>
        <p class="text-slate-500 text-sm mt-1">管理用于接入网关的逻辑凭证与配额</p>
      </div>
      <button 
        @click="showCreateModal = true"
        class="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
      >
        <span class="text-lg">+</span> {{ $t('keys.generate') }}
      </button>
    </div>

    <div class="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <table class="w-full text-left border-collapse">
        <thead class="bg-slate-50 border-b border-slate-200">
          <tr>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">ID</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.value') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.limit') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.used') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.status') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase text-right">{{ $t('keys.table.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200">
          <tr v-for="key in keys" :key="key.id" class="hover:bg-slate-50 transition-colors">
            <td class="px-6 py-4 text-sm text-slate-600">#{{ key.id }}</td>
            <td class="px-6 py-4">
              <div class="flex items-center gap-2">
                <code class="bg-slate-100 px-2 py-1 rounded text-indigo-600 font-mono text-xs">
                  {{ maskKey(key.key_value) }}
                </code>
                <button @click="copyToClipboard(key.key_value)" class="text-slate-400 hover:text-indigo-600 transition-colors">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
              </div>
            </td>
            <td class="px-6 py-4 text-sm text-slate-600">
              {{ key.daily_limit === -1 ? $t('keys.table.limitUnlimited') : key.daily_limit.toLocaleString() }}
            </td>
            <td class="px-6 py-4 text-sm text-slate-600">{{ key.used_tokens.toLocaleString() }}</td>
            <td class="px-6 py-4">
              <span 
                class="px-2 py-1 rounded-full text-xs font-medium"
                :class="isExhausted(key) ? 'bg-red-50 text-red-600' : 'bg-emerald-50 text-emerald-600'"
              >
                {{ isExhausted(key) ? $t('keys.table.statusExhausted') : $t('keys.table.statusActive') }}
              </span>
            </td>
            <td class="px-6 py-4 text-right">
              <button 
                @click="confirmDelete(key.id)"
                class="text-slate-400 hover:text-red-600 transition-colors text-sm font-medium"
              >
                {{ $t('keys.revoke') }}
              </button>
            </td>
          </tr>
          <tr v-if="keys.length === 0">
            <td colspan="6" class="px-6 py-12 text-center text-slate-400">暂无网关密钥</td>
          </tr>
        </tbody>
      </table>
    </div>

    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { GatewayKeysAPI } from '../api'

const { t } = useI18n()
const keys = ref([])
const showCreateModal = ref(false)

const fetchKeys = async () => {
  try {
    keys.value = await GatewayKeysAPI.list()
  } catch (err) {
    console.error('Failed to load keys')
  }
}

const maskKey = (val) => {
  if (!val) return ''
  return `${val.substring(0, 8)}****************${val.substring(val.length - 4)}`
}

const isExhausted = (key) => {
  return key.daily_limit !== -1 && key.used_tokens >= key.daily_limit
}

const copyToClipboard = (val) => {
  navigator.clipboard.writeText(val)
  // 此处可集成一个简单的 Toast 提示
}

const confirmDelete = async (id) => {
  if (confirm(t('keys.revokeConfirm'))) {
    try {
      await GatewayKeysAPI.delete(id)
      await fetchKeys()
    } catch (err) {
      alert('撤销失败')
    }
  }
}

onMounted(fetchKeys)
</script>