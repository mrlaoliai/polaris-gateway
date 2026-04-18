<template>
  <div :class="{ 'dark': isDark }" class="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-950">
    <aside class="w-64 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col">
      <div class="h-16 flex items-center px-6 border-b border-slate-100 dark:border-slate-800">
        <span class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">🛰️ Polaris <span class="text-blue-600">v2.0</span></span>
      </div>

      <nav class="flex-1 overflow-y-auto py-4 px-3 space-y-1">
        <router-link to="/" class="nav-item" active-class="nav-active" exact>{{ $t('nav.dashboard') }}</router-link>

        <div class="pt-2 pb-1 px-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">代理</div>
        <router-link to="/keys" class="nav-item" active-class="nav-active">{{ $t('nav.gateway_keys') }}</router-link>
        <router-link to="/routing" class="nav-item" active-class="nav-active">{{ $t('nav.routing_rules') }}</router-link>
        <router-link to="/accounts" class="nav-item" active-class="nav-active">{{ $t('nav.accounts') }}</router-link>
        <router-link to="/user-providers" class="nav-item" active-class="nav-active">{{ $t('nav.user_providers') }}</router-link>

        <div class="pt-2 pb-1 px-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">配置</div>
        <router-link to="/providers" class="nav-item" active-class="nav-active">{{ $t('nav.providers') }}</router-link>
      </nav>

      <div class="p-4 border-t border-slate-100 dark:border-slate-800 text-xs text-slate-400 text-center">Powered by mrlaoliai</div>
    </aside>

    <div class="flex-1 flex flex-col relative overflow-hidden">
      <header class="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between px-8 z-10">
        <h1 class="text-lg font-semibold text-slate-800 dark:text-slate-100">{{ routeTitle }}</h1>

        <div class="flex items-center space-x-4">
          <button @click="toggleTheme" class="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors">
            {{ isDark ? '🌙' : '☀️' }}
          </button>
          <div class="flex items-center space-x-1 bg-slate-100 dark:bg-slate-800 p-1 rounded-md border border-slate-200 dark:border-slate-700">
            <button v-for="l in ['zh', 'en']" :key="l" @click="toggleLanguage(l)" :class="['px-3 py-1 text-sm rounded transition-all', currentLocale === l ? 'bg-white dark:bg-slate-700 shadow-sm font-medium text-slate-800 dark:text-slate-100' : 'text-slate-500']">{{ l === 'zh' ? '中' : 'EN' }}</button>
          </div>
        </div>
      </header>

      <main class="flex-1 overflow-y-auto p-8"><router-view /></main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { toggleLanguage } from '../i18n'

const { locale, t } = useI18n()
const currentLocale = computed(() => locale.value)
const route = useRoute()

// 路由名称到 i18n key 的映射表，避免 toLowerCase 导致多词名出错
const routeNameMap = {
  Dashboard: 'nav.dashboard',
  GatewayKeys: 'nav.gateway_keys',
  RoutingRules: 'nav.routing_rules',
  Accounts: 'nav.accounts',
  UserProviders: 'nav.user_providers',
  Providers: 'nav.providers'
}
const routeTitle = computed(() => {
  const key = routeNameMap[route.name]
  return key ? t(key) : ''
})
const isDark = ref(localStorage.getItem('polaris-theme') === 'dark')

const toggleTheme = () => {
  isDark.value = !isDark.value
  localStorage.setItem('polaris-theme', isDark.value ? 'dark' : 'light')
}
</script>

<style scoped>
.nav-item { @apply block px-3 py-2 rounded-md text-sm text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100 transition-colors; }
.nav-active { @apply bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 font-medium; }
</style>