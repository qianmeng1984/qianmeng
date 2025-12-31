import axios from 'axios'
import { ElMessage } from 'element-plus'

// 创建 axios 实例
const service = axios.create({
    baseURL: 'http://localhost:8080/api/v1', // 指向你的 Go 后端
    timeout: 60000 // 请求超时时间
})

// 请求拦截器：每次请求自动带上 Token
service.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token')
        if (token) {
            config.headers['Authorization'] = `Bearer ${token}`
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// 响应拦截器
service.interceptors.response.use(
    response => {
        return response.data
    },
    error => {
        // 【核心修复】如果是登录接口(/login)报 401，不要全局拦截，直接抛出给 Login.vue 处理
        if (error.response && error.response.status === 401) {
            const url = error.config.url || ''
            if (url.includes('/login')) {
                return Promise.reject(error) // 抛出去，让登录页弹窗 "密码错误"
            } else {
                ElMessage.error('登录已过期，请重新登录')
            }
        } else {
            ElMessage.error(error.response?.data || '网络请求错误')
        }
        return Promise.reject(error)
    }
)
export default service