<template>
  <div class="max-w-6xl mx-auto space-y-6">
    <div class="flex justify-between items-center mb-8">
      <div>
        <h2 class="text-2xl font-bold text-slate-800">{{ $t('nav.routing_rules') }}</h2>
        <p class="text-slate-500 text-sm mt-1">Configure Smart Router mappings and dynamic DSL transformation rules.</p>
      </div>
      <button 
        @click="showCreateModal = true"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center"
      >
        <svg class="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ $t('common.add') }} Rule
      </button>
    </div>

    <div class="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="bg-slate-50 text-slate-500 border-b border-slate-100">
            <tr>
              <th class="px-6 py-3 font-medium">Virtual In-Model</th>
              <th class="px-6 py-3 font-medium">Physical Target</th>
              <th class="px-6 py-3 font-medium">Fallback Target</th>
              <th class="px-6 py-3 font-medium">DSL Active</th>
              <th class="px-6 py-3 font-medium text-right">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="rule in rules" :key="rule.id" class="hover:bg-slate-50/50 transition-colors">
              <td class="px-6 py-4 font-medium text-slate-700">{{ rule.in_model }}</td>
              <td class="px-6 py-4">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100">
                  {{ rule.target_model }}
                </span>
              </td>
              <td class="px-6 py-4 text-slate-500">
                <span v-if="rule.fallback_model" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 border border-slate-200">
                  {{ rule.fallback_model }}
                </span>
                <span v-else class="text-slate-300">-</span>
              </td>
              <td class="px-6 py-4">
                <span v-if="rule.has_dsl" class="inline-flex items-center text-xs text-emerald-600 font-medium">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  CEL-go
                </span>
                <span v-else class="text-slate-400 text-xs">Standard</span>
              </td>
              <td class="px-6 py-4 text-right">
                <button @click="deleteRule(rule.id)" class="text-red-500 hover:text-red-700 text-sm font-medium transition-colors">
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

const rules = ref([])
const showCreateModal = ref(false)

const loadRules = () => {
  setTimeout(() => {
    rules.value = [
      { id: 1, in_model: 'claude-3-opus', target_model: 'deepseek-v4-reasoning', fallback_model: 'gemini-1.5-pro', has_dsl: true },
      { id: 2, in_model: 'claude-3-5-sonnet', target_model: 'gemini-1.5-pro', fallback_model: null, has_dsl: false },
      { id: 3, in_model: 'gpt-4o', target_model: 'gemini-1.5-pro', fallback_model: 'deepseek-v4-reasoning', has_dsl: true }
    ]
  }, 200)
}

const deleteRule = (id) => {
  rules.value = rules.value.filter(r => r.id !== id)
}

onMounted(() => {
  loadRules()
})
</script>