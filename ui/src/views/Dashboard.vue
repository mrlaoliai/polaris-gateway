<template>
  <div class="max-w-6xl mx-auto space-y-6">
    <div class="mb-8">
      <h2 class="text-2xl font-bold text-slate-800">Polaris Gateway Node</h2>
      <p class="text-slate-500 text-sm mt-1">Autonomous AI Protocol Orchestrator (Zero-CGO & State-in-DB)</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      
      <div class="bg-white rounded-xl border border-slate-200 p-6 shadow-sm flex flex-col">
        <div class="text-sm font-medium text-slate-500 mb-2">{{ $t('dashboard.total_tokens') }}</div>
        <div class="text-3xl font-bold text-slate-800 tracking-tight">
          {{ stats.totalTokens.toLocaleString() }}
        </div>
        <div class="mt-auto pt-4 flex items-center text-xs text-green-600 font-medium">
          <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
          </svg>
          +12% from yesterday
        </div>
      </div>

      <div class="bg-white rounded-xl border border-slate-200 p-6 shadow-sm flex flex-col">
        <div class="text-sm font-medium text-slate-500 mb-2">{{ $t('dashboard.active_agents') }}</div>
        <div class="text-3xl font-bold text-slate-800 tracking-tight">
          {{ stats.activeAgents }}
        </div>
        <div class="mt-auto pt-4 text-xs text-slate-400">
          Connected via Bifrost 2.0
        </div>
      </div>

      <div class="bg-white rounded-xl border border-slate-200 p-6 shadow-sm flex flex-col">
        <div class="text-sm font-medium text-slate-500 mb-2">{{ $t('dashboard.sentinel_health') }}</div>
        <div class="flex items-end space-x-2">
          <span class="text-3xl font-bold text-emerald-600 tracking-tight">Healthy</span>
          <span class="text-sm text-slate-500 mb-1">100%</span>
        </div>
        <div class="mt-auto pt-4 text-xs text-slate-400">
          Managed by Sentinel Orchestrator
        </div>
      </div>
    </div>

    <div class="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden mt-8">
      <div class="px-6 py-4 border-b border-slate-100 flex justify-between items-center">
        <h3 class="font-semibold text-slate-800">Recent Routing Traces</h3>
        <span class="text-xs text-slate-400 bg-slate-100 px-2 py-1 rounded">Live DB Sync</span>
      </div>
      
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="bg-slate-50 text-slate-500 border-b border-slate-100">
            <tr>
              <th class="px-6 py-3 font-medium">Trace ID</th>
              <th class="px-6 py-3 font-medium">Virtual Model</th>
              <th class="px-6 py-3 font-medium">Physical Target</th>
              <th class="px-6 py-3 font-medium">Latency</th>
              <th class="px-6 py-3 font-medium">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="trace in recentTraces" :key="trace.id" class="hover:bg-slate-50/50 transition-colors">
              <td class="px-6 py-4 font-mono text-xs text-slate-500">{{ trace.id.substring(0, 8) }}</td>
              <td class="px-6 py-4 font-medium text-slate-700">{{ trace.inModel }}</td>
              <td class="px-6 py-4 text-slate-600">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100">
                  {{ trace.outModel }}
                </span>
              </td>
              <td class="px-6 py-4 text-slate-500">{{ trace.latency }}ms</td>
              <td class="px-6 py-4">
                <span class="inline-block w-2 h-2 rounded-full" :class="trace.status === 'success' ? 'bg-emerald-400' : 'bg-red-400'"></span>
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
import { DashboardAPI } from '../api'

// 响应式状态定义
const stats = ref({
  totalTokens: 0,
  activeAgents: 0,
  health: 'Loading...'
})
const recentTraces = ref([])

// 初始化数据拉取（真实 API）
onMounted(async () => {
  try {
    const overview = await DashboardAPI.getStats()
    stats.value = overview
  } catch (e) {
    console.error('获取概览数据失败', e)
  }
  try {
    const traces = await DashboardAPI.getRecentTraces()
    recentTraces.value = traces.map(t => ({
      id: t.id,
      inModel: '-',
      outModel: '-',
      latency: t.latency,
      status: t.status
    }))
  } catch (e) {
    console.error('获取 Trace 数据失败', e)
  }
})
</script>