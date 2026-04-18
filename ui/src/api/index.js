// ui/src/api/index.js
import axios from 'axios'

const apiClient = axios.create({
    baseURL: '/api/v1',
    timeout: 5000,
    headers: {
        'Content-Type': 'application/json'
    }
})

apiClient.interceptors.response.use(
    response => response.data,
    error => {
        console.error('API Error:', error.response?.data?.message || error.message)
        return Promise.reject(error)
    }
)

export const DashboardAPI = {
    getStats: () => apiClient.get('/stats/overview'),
    getRecentTraces: () => apiClient.get('/stats/traces?limit=5')
}

export const GatewayKeysAPI = {
    list: () => apiClient.get('/keys'),
    create: (data) => apiClient.post('/keys', data),
    delete: (id) => apiClient.delete(`/keys/${id}`)
}

// [新增] 补全物理账号与路由规则 API
export const AccountsAPI = {
    list: () => apiClient.get('/accounts'),
    create: (data) => apiClient.post('/accounts', data),
    delete: (id) => apiClient.delete(`/accounts/${id}`)
}

export const RoutingRulesAPI = {
    list: () => apiClient.get('/routing'),
    create: (data) => apiClient.post('/routing', data),
    delete: (id) => apiClient.delete(`/routing/${id}`)
}