// ui/src/api/index.js
import axios from 'axios'

// 创建 axios 实例，由于是同一二进制部署，默认使用相对路径
const apiClient = axios.create({
    baseURL: '/api/v1',
    timeout: 5000,
    headers: {
        'Content-Type': 'application/json'
    }
})

// 响应拦截器：统一处理后端报错
apiClient.interceptors.response.use(
    response => response.data,
    error => {
        // 可以在这里接入全局的 Toast 提示组件
        console.error('API Error:', error.response?.data?.message || error.message)
        return Promise.reject(error)
    }
)

// 导出与后端对应的 API 服务
export const DashboardAPI = {
    getStats: () => apiClient.get('/stats/overview'), // 获取总览数据
    getRecentTraces: () => apiClient.get('/stats/traces?limit=5') // 获取最近审计日志
}

export const GatewayKeysAPI = {
    list: () => apiClient.get('/keys'),
    create: (data) => apiClient.post('/keys', data),
    delete: (id) => apiClient.delete(`/keys/${id}`)
}