import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from 'src/api/client'

export const useWalletStore = defineStore('wallet', () => {
  const account = ref(null)
  const loading = ref(false)
  const error = ref(null)

  // 创建账户
  async function createAccount(payload) {
    const data = await apiClient.post('/account/create', payload)
    account.value = data.account || data
    return account.value
  }

  // 获取账户信息
  async function fetchAccount() {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get('/account')
      // 支持 { account: {...} }, { accounts: [...] }, { Accounts: [...] } 三种响应格式
      const accounts = data.accounts || data.Accounts
      if (Array.isArray(accounts)) {
        account.value = accounts[0] || null
      } else {
        account.value = data.account || data
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  // 充值（跳转支付宝）
  async function topUp(accountId, amount) {
    const data = await apiClient.post('/payments/create', { account_id: accountId, amount })
    // data.pay_url 是支付宝收银台 URL
    if (data.pay_url) {
      window.location.href = data.pay_url
    }
    return data
  }

  // 提现申请
  async function withdraw(payload) {
    const data = await apiClient.post('/withdrawals/create', payload)
    return data.withdrawal || data
  }

  return { account, loading, error, createAccount, fetchAccount, topUp, withdraw }
})
