<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">{{ $t('nav.gateway_keys') }}</h1>
        <p class="text-slate-500 text-sm mt-1">管理用于接入网关的逻辑凭证与配额</p>
      </div>
      <button @click="showCreateModal = true" class="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2">
        <span class="text-lg">+</span> {{ $t('keys.generate') }}
      </button>
    </div>

    <div class="p-card overflow-hidden">
      <table class="w-full text-left border-collapse">
        <thead class="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
          <tr>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">ID</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.value') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.limit') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase">{{ $t('keys.table.used') }}</th>
            <th class="px-6 py-4 text-xs font-semibold text-slate-500 uppercase text-right">{{ $t('keys.table.actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200 dark:divide-slate-800">
          <tr v-for="key in keys" :key="key.id" class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
            <td class="px-6 py-4 text-sm">#{{ key.id }}</td>
            <td class="px-6 py-4 font-mono text-xs">{{ maskKey(key.key_value) }}</td>
            <td class="px-6 py-4 text-sm">{{ key.daily_limit === -1 ? '∞' : key.daily_limit }}</td>
            <td class="px-6 py-4 text-sm">{{ key.used_tokens }}</td>
            <td class="px-6 py-4 text-right">
              <button @click="confirmDelete(key.id)" class="text-red-500 hover:text-red-700 text-sm font-medium">{{ $t('common.delete') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-sm">
      <div class="p-card w-full max-w-md p-6 space-y-4">
        <h3 class="text-lg font-bold">{{ $t('keys.modal.title') }}</h3>
        <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{{ $t('keys.table.limit') }}</label>
            <input v-model.number="newKeyLimit" type="number" class="p-input" />
        </div>
        <div class="flex justify-end space-x-3 pt-4">
          <button @click="showCreateModal = false" class="px-4 py-2 text-sm text-slate-600 dark:text-slate-400">{{ $t('common.cancel') }}</button>
          <button @click="handleCreate" class="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium">{{ $t('common.confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GatewayKeysAPI } from '../api'

const keys = ref([])
const showCreateModal = ref(false)
const newKeyLimit = ref(-1)

const fetchKeys = async () => {
    try { keys.value = await GatewayKeysAPI.list() } catch (e) {}
}

const handleCreate = async () => {
    try {
        await GatewayKeysAPI.create({ daily_limit: newKeyLimit.value })
        showCreateModal.value = false
        await fetchKeys()
    } catch (e) {}
}

const confirmDelete = async (id) => {
    if (confirm('确认撤销？')) {
        await GatewayKeysAPI.delete(id)
        await fetchKeys()
    }
}

const maskKey = (val) => val ? `${val.substring(0, 8)}...${val.substring(val.length - 4)}` : ''

onMounted(fetchKeys)
</script>