import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from 'src/api/client'

export const useTransactionStore = defineStore('transaction', () => {
  const transactions = ref([])
  const loading = ref(false)
  const error = ref(null)
  const totalCount = ref(0)
  const page = ref(1)
  const pageSize = ref(10)

  // Total income (positive amounts, excluding refunds)
  const totalIncome = computed(() =>
    transactions.value
      .filter(t => t.direction === 'income' &&
                   t.tradeType !== 'BOUNTY_REFUND' &&
                   t.tradeType !== 'WITHDRAWAL_REFUND')
      .reduce((sum, t) => sum + Number(t.amount), 0)
  )

  // Total expense (negative amounts / frozen)
  const totalExpense = computed(() =>
    transactions.value
      .filter(t => t.direction === 'expense')
      .reduce((sum, t) => sum + Number(t.amount), 0)
  )

  // Get direction based on trade_type and current account
  function getDirection(transfer, currentAccountId) {
    const { from_account_id, to_account_id, trade_type } = transfer

    // Handle special trade types first
    if (trade_type === 'WITHDRAWAL' || trade_type === 'WITHDRAWAL_OUT') {
      // Withdrawal is always expense for the user
      return 'expense'
    }

    if (trade_type === 'WITHDRAWAL_REFUND' || trade_type === 'BOUNTY_REFUND') {
      // Refunds should be treated as unfrozen (解冻), not income
      return 'frozen'
    }

    // Internal transfers (from == to): BOUNTY_FREEZE
    if (from_account_id === to_account_id) {
      if (trade_type === 'BOUNTY_FREEZE') return 'frozen'
      return 'frozen' // default internal
    }

    // Cross-account transfers
    if (String(from_account_id) === String(currentAccountId)) {
      return 'expense'
    }
    if (String(to_account_id) === String(currentAccountId)) {
      return 'income'
    }
    return 'expense'
  }

  // Map trade_type to display label
  const TRADE_TYPE_LABELS = {
    TRANSFER: '转账',
    DEPOSIT: '充值',
    WITHDRAWAL: '提现',
    WITHDRAWAL_OUT: '提现',
    WITHDRAWAL_REFUND: '提现解冻',
    BOUNTY_FREEZE: '冻结',
    BOUNTY_PAYOUT: '悬赏支出',
    BOUNTY_REFUND: '解冻',
  }

  // Normalize a raw transfer record
  function normalizeTransfer(raw, currentAccountId) {
    const direction = getDirection(raw, currentAccountId)
    const displayType = TRADE_TYPE_LABELS[raw.trade_type] || raw.trade_type || '转账'

    return {
      id: raw.id,
      direction,
      displayType,
      amount: Number(raw.amount),
      description: raw.description || '',
      tradeType: raw.trade_type,
      fromAccountId: raw.from_account_id,
      toAccountId: raw.to_account_id,
      createdAt: raw.created_at || new Date().toISOString(),
      status: 'completed',
    }
  }

  async function fetchAll(accountId) {
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.get('/transfers', {
        params: { account_id: accountId, page: page.value, page_size: pageSize.value },
      })

      // Handle different response shapes
      const data = response.transfers || response.data || response
      const list = Array.isArray(data) ? data : []

      totalCount.value = Number(response.total || list.length)

      const normalized = list.map(raw => normalizeTransfer(raw, accountId))

      // Sort by created_at descending
      normalized.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))

      if (page.value === 1) {
        transactions.value = normalized
      } else {
        transactions.value.push(...normalized)
      }

      return transactions.value
    } catch (e) {
      error.value = e.message || '获取交易记录失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchTransactions(accountId) {
    return fetchAll(accountId)
  }

  async function fetchTransactionsByDateRange(accountId) {
    page.value = 1
    return fetchAll(accountId)
  }

  function setPage(n) {
    page.value = n
  }

  function reset() {
    transactions.value = []
    page.value = 1
    error.value = null
    totalCount.value = 0
  }

  return {
    transactions,
    loading,
    error,
    totalCount,
    page,
    pageSize,
    totalIncome,
    totalExpense,
    fetchTransactions,
    fetchTransactionsByDateRange,
    setPage,
    fetchAll,
    reset,
  }
})
