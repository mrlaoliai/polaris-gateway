<template>
  <div class="max-w-6xl mx-auto space-y-6">
    <div class="flex justify-between items-center mb-8">
      <div>
        <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">{{ $t('nav.routing_rules') }}</h2>
        <p class="text-slate-500 text-sm mt-1">Configure Smart Router mappings and dynamic DSL transformation rules.</p>
      </div>
      <button
        @click="openCreateModal"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm flex items-center"
      >
        <svg class="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ $t('common.add') }} Rule
      </button>
    </div>

    <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead class="bg-slate-50 dark:bg-slate-800/50 text-slate-500 border-b border-slate-100 dark:border-slate-800">
            <tr>
              <th class="px-6 py-3 font-medium">Virtual In-Model</th>
              <th class="px-6 py-3 font-medium">Physical Target</th>
              <th class="px-6 py-3 font-medium">Fallback Target</th>
              <th class="px-6 py-3 font-medium">DSL Active</th>
              <th class="px-6 py-3 font-medium text-right">{{ $t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800">
            <tr v-if="rules.length === 0">
              <td colspan="5" class="px-6 py-12 text-center text-slate-400 text-sm">暂无路由规则，点击右上角添加</td>
            </tr>
            <tr v-for="rule in rules" :key="rule.id" class="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
              <td class="px-6 py-4 font-medium text-slate-700 dark:text-slate-200">{{ rule.in_model }}</td>
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

    <!-- 创建路由规则弹窗 -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm">
      <div class="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-xl w-full max-w-lg p-6 space-y-5">
        <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">添加路由规则</h3>

        <div class="space-y-4">
          <!-- 虚拟入模型名（客户端请求时使用的模型名） -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Virtual In-Model <span class="text-red-500">*</span>
            </label>
            <input
              v-model="form.inModel"
              type="text"
              placeholder="如: claude-3-opus、gpt-4o"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm"
            />
            <p class="text-xs text-slate-400 mt-1">客户端请求时携带的虚拟模型名，由网关映射到物理模型</p>
          </div>

          <!-- 目标 Provider -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Target Provider <span class="text-red-500">*</span>
            </label>
            <select
              v-model.number="form.targetProviderId"
              @change="onTargetProviderChange"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm"
            >
              <option value="" disabled>-- 选择目标厂商 --</option>
              <option v-for="p in providers" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.protocol_type }})
              </option>
            </select>
          </div>

          <!-- 目标模型 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Target Model <span class="text-red-500">*</span>
            </label>
            <select
              v-model.number="form.targetSpecId"
              :disabled="targetSpecs.length === 0"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm disabled:opacity-50"
            >
              <option value="" disabled>{{ form.targetProviderId ? '-- 选择目标模型 --' : '请先选择 Provider' }}</option>
              <option v-for="s in targetSpecs" :key="s.id" :value="s.id">
                {{ s.model_name }}
                <template v-if="s.supports_thinking"> 🧠</template>
                <template v-if="s.supports_vision"> 👁️</template>
              </option>
            </select>
          </div>

          <!-- Fallback Provider（可选） -->
          <div>
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Fallback Provider <span class="text-slate-400 font-normal">(可选)</span>
            </label>
            <select
              v-model.number="form.fallbackProviderId"
              @change="onFallbackProviderChange"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm"
            >
              <option value="">-- 不设置备用 --</option>
              <option v-for="p in providers" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.protocol_type }})
              </option>
            </select>
          </div>

          <!-- Fallback 模型（可选） -->
          <div v-if="form.fallbackProviderId">
            <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Fallback Model
            </label>
            <select
              v-model.number="form.fallbackSpecId"
              :disabled="fallbackSpecs.length === 0"
              class="w-full px-3 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md focus:ring-2 focus:ring-blue-500 outline-none transition-all text-sm disabled:opacity-50"
            >
              <option value="" disabled>-- 选择备用模型 --</option>
              <option v-for="s in fallbackSpecs" :key="s.id" :value="s.id">
                {{ s.model_name }}
              </option>
            </select>
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
import { RoutingRulesAPI } from '../api'
import axios from 'axios'

const rules = ref([])
const providers = ref([])
const targetSpecs = ref([])
const fallbackSpecs = ref([])
const showCreateModal = ref(false)
const submitting = ref(false)
const formError = ref('')

const form = ref({
  inModel: '',
  targetProviderId: '',
  targetSpecId: '',
  fallbackProviderId: '',
  fallbackSpecId: ''
})

const resetForm = () => {
  form.value = {
    inModel: '',
    targetProviderId: '',
    targetSpecId: '',
    fallbackProviderId: '',
    fallbackSpecId: ''
  }
  targetSpecs.value = []
  fallbackSpecs.value = []
  formError.value = ''
}

const openCreateModal = async () => {
  try {
    const res = await axios.get('/api/v1/providers')
    providers.value = res.data
  } catch (e) {
    providers.value = []
  }
  resetForm()
  showCreateModal.value = true
}

const closeModal = () => {
  showCreateModal.value = false
  resetForm()
}

// Target Provider 变化时，级联加载对应模型列表
const onTargetProviderChange = async () => {
  form.value.targetSpecId = ''
  targetSpecs.value = []
  if (!form.value.targetProviderId) return
  try {
    const res = await axios.get(`/api/v1/model-specs?provider_id=${form.value.targetProviderId}`)
    targetSpecs.value = res.data
  } catch (e) {
    targetSpecs.value = []
  }
}

// Fallback Provider 变化时，级联加载备用模型列表
const onFallbackProviderChange = async () => {
  form.value.fallbackSpecId = ''
  fallbackSpecs.value = []
  if (!form.value.fallbackProviderId) return
  try {
    const res = await axios.get(`/api/v1/model-specs?provider_id=${form.value.fallbackProviderId}`)
    fallbackSpecs.value = res.data
  } catch (e) {
    fallbackSpecs.value = []
  }
}

const loadRules = async () => {
  try {
    rules.value = await RoutingRulesAPI.list()
  } catch (e) {
    console.error('获取路由规则失败', e)
  }
}

const handleCreate = async () => {
  formError.value = ''
  if (!form.value.inModel.trim()) {
    formError.value = 'Virtual In-Model 不能为空'
    return
  }
  if (!form.value.targetSpecId) {
    formError.value = '请选择 Target Model'
    return
  }
  submitting.value = true
  try {
    await RoutingRulesAPI.create({
      in_model: form.value.inModel.trim(),
      target_spec_id: form.value.targetSpecId,
      fallback_spec_id: form.value.fallbackSpecId || undefined
    })
    closeModal()
    await loadRules()
  } catch (e) {
    formError.value = '创建失败，请检查参数'
    console.error(e)
  } finally {
    submitting.value = false
  }
}

const deleteRule = async (id) => {
  if (confirm('确认删除该路由规则？')) {
    await RoutingRulesAPI.delete(id)
    await loadRules()
  }
}

onMounted(loadRules)
</script>