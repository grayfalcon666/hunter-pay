import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// 请求拦截器：附加 Token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    console.log('[apiClient] request:', config.method?.toUpperCase(), config.baseURL + config.url, 'params:', JSON.stringify(config.params), 'data:', JSON.stringify(config.data))
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：处理 401，统一错误格式
apiClient.interceptors.response.use(
  (response) => {
    console.log('[apiClient] response:', response.config.url, response.status, JSON.stringify(response.data))
    return response.data
  },
  (error) => {
    console.log('[apiClient] error response:', error.response?.config?.url, error.response?.status, JSON.stringify(error.response?.data))
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      window.location.href = '/#/login'
    }

    const message =
      error.response?.data?.error ||
      error.response?.data?.message ||
      error.message ||
      '网络错误，请稍后重试'

    return Promise.reject({ message, status: error.response?.status })
  }
)

export default apiClient
