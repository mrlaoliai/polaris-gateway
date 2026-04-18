// ui/src/router/index.js
import { createRouter, createWebHashHistory } from 'vue-router'
import Layout from '../components/Layout.vue'

const Dashboard = () => import('../views/Dashboard.vue')
const GatewayKeys = () => import('../views/GatewayKeys.vue')
const RoutingRules = () => import('../views/RoutingRules.vue')
const Accounts = () => import('../views/Accounts.vue')

export const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        {
            path: '/',
            component: Layout,
            children: [
                { path: '', redirect: '/dashboard' },
                { path: 'dashboard', name: 'Dashboard', component: Dashboard },
                { path: 'keys', name: 'GatewayKeys', component: GatewayKeys },
                { path: 'routing', name: 'RoutingRules', component: RoutingRules },
                { path: 'accounts', name: 'Accounts', component: Accounts }
            ]
        }
    ]
})