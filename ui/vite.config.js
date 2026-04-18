// ui/vite.config.js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
    plugins: [vue()],
    base: '/dashboard/',
    resolve: {
        alias: {
            '@': resolve(__dirname, 'src')
        }
    },
    build: {
        outDir: 'dist',
        emptyOutDir: true,
        // 确保打包出的资源路径是相对的，方便 go:embed 后的路径映射
        base: './'
    }
})