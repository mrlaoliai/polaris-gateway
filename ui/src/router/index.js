// ui/src/router/index.js
// 作者：mrlaoliai
// 设计哲学：极简路由配置，消除冗余重定向，适配嵌入式 VFS 环境
import { createRouter, createWebHashHistory } from 'vue-router'
import Layout from '../components/Layout.vue'

// 路由组件懒加载，优化首屏加载速度
const Dashboard = () => import('../views/Dashboard.vue')
const GatewayKeys = () => import('../views/GatewayKeys.vue')
const RoutingRules = () => import('../views/RoutingRules.vue')
const Accounts = () => import('../views/Accounts.vue')
const Providers = () => import('../views/Providers.vue')

export const router = createRouter({
    // 使用 Hash 模式以获得最佳的后端兼容性（无需复杂的 Nginx/Go 转发配置）
    history: createWebHashHistory(),
    routes: [
        {
            path: '/',
            component: Layout,
            children: [
                // 核心修改：将默认空路径直接指向 Dashboard 组件
                // 这样访问 /dashboard/ 时，URL 会保持为 /#/ 而不会跳转到 /#/dashboard
                {
                    path: '',
                    name: 'Dashboard',
                    component: Dashboard
                },
                {
                    path: 'keys',
                    name: 'GatewayKeys',
                    component: GatewayKeys
                },
                {
                    path: 'routing',
                    name: 'RoutingRules',
                    component: RoutingRules
                },
                {
                    path: 'accounts',
                    name: 'Accounts',
                    component: Accounts
                },
                {
                    path: 'providers',
                    name: 'Providers',
                    component: Providers
                }
            ]
        }
    ]
})