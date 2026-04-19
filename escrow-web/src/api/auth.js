import apiClient from './client'

// 注册
export async function register(payload) {
  return apiClient.post('/auth/register', payload)
}

// 登录
export async function login(payload) {
  return apiClient.post('/auth/login', payload)
}

// 邮箱验证
export async function verifyEmail(params) {
  return apiClient.get('/auth/verify_email', { params })
}
