// ui/src/i18n.js
import { createI18n } from 'vue-i18n'
import zh from './locales/zh.json'
import en from './locales/en.json'

// 尝试从本地存储获取语言偏好，默认中文
const savedLang = localStorage.getItem('polaris-lang') || 'zh'

export const i18n = createI18n({
    legacy: false, // 启用 Vue 3 Composition API 模式
    locale: savedLang,
    fallbackLocale: 'en',
    messages: {
        zh,
        en
    }
})

// 提供一个全局切换函数
export function toggleLanguage(targetLang) {
    i18n.global.locale.value = targetLang
    localStorage.setItem('polaris-lang', targetLang)
}