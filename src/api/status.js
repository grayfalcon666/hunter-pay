import axios from './axios'

/**
 * 获取用户初始化状态
 * @param {string} username - 用户名
 * @returns {Promise} 返回用户状态信息
 */
export function getUserStatus(username) {
  return axios.get(`/auth/status/${username}`)
}

/**
 * 更新用户状态（内部使用）
 * @param {string} username - 用户名
 * @param {string} status - 状态
 * @returns {Promise}
 */
export function updateUserStatus(username, status) {
  return axios.post(`/auth/status/${username}`, { status })
}
