<template>
  <div class="flex h-screen overflow-hidden bg-slate-50">
    
    <aside class="w-64 bg-white border-r border-slate-200 flex flex-col">
      <div class="h-16 flex items-center px-6 border-b border-slate-100">
        <span class="text-xl font-bold text-slate-800 tracking-tight">🛰️ Polaris <span class="text-blue-600">v2.0</span></span>
      </div>

      <nav class="flex-1 overflow-y-auto py-4 px-3 space-y-1">
        <router-link 
          to="/dashboard" 
          class="nav-item" 
          active-class="bg-blue-50 text-blue-700 font-medium"
        >
          {{ $t('nav.dashboard') }}
        </router-link>
        
        <router-link 
          to="/keys" 
          class="nav-item" 
          active-class="bg-blue-50 text-blue-700 font-medium"
        >
          {{ $t('nav.gateway_keys') }}
        </router-link>
        
        <div class="nav-item cursor-not-allowed opacity-50">{{ $t('nav.routing_rules') }}</div>
        <div class="nav-item cursor-not-allowed opacity-50">{{ $t('nav.accounts') }}</div>
      </nav>

      <div class="p-4 border-t border-slate-100 text-xs text-slate-400 text-center">
        Powered by mrlaoliai
      </div>
    </aside>

    <div class="flex-1 flex flex-col relative overflow-hidden">
      <header class="h-16 bg-white border-b border-slate-200 flex items-center justify-between px-8 z-10">
        <h1 class="text-lg font-semibold text-slate-800">
          {{ $route.name ? $t('nav.' + $route.name.toLowerCase()) : '' }}
        </h1>

        <div class="flex items-center space-x-2 bg-slate-100 p-1 rounded-md border border-slate-200">
          <button 
            @click="switchLang('zh')"
            :class="['px-3 py-1 text-sm rounded transition-colors', currentLang === 'zh' ? 'bg-white shadow-sm font-medium text-slate-800' : 'text-slate-500 hover:text-slate-700']"
          >
            中
          </button>
          <button 
            @click="switchLang('en')"
            :class="['px-3 py-1 text-sm rounded transition-colors', currentLang === 'en' ? 'bg-white shadow-sm font-medium text-slate-800' : 'text-slate-500 hover:text-slate-700']"
          >
            EN
          </button>
        </div>
      </header>

      <main class="flex-1 overflow-y-auto p-8">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { toggleLanguage } from '../i18n'

const { locale } = useI18n()
const currentLang = computed(() => locale.value)

const switchLang = (lang) => {
  toggleLanguage(lang)
}
</script>

<style scoped>
/* 导航项基础样式 */
.nav-item {
  @apply block px-3 py-2 rounded-md text-sm text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-colors;
}

/* 简单的路由切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>