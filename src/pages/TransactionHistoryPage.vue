<template>
  <q-page class="history-page">
    <q-pull-to-refresh @refresh="onRefresh" color="accent-gold">
      <div class="page-inner">
        <!-- Page Header -->
        <div class="page-header">
          <button class="back-btn" @click="$router.push('/wallet')">
            <q-icon name="arrow_back" size="18px" />
          </button>
          <h1 class="page-title">资金明细</h1>
        </div>

        <!-- Summary Stats -->
        <div class="summary-row">
          <div class="stat-card stat-card--income">
            <div class="stat-icon">
              <q-icon name="trending_up" size="18px" />
            </div>
            <div class="stat-body">
              <span class="stat-label">收入总额</span>
              <span class="stat-value stat-value--income">
                +¥{{ formatYuan(transactionStore.totalIncome) }}
              </span>
            </div>
          </div>

          <div class="stat-card stat-card--expense">
            <div class="stat-icon">
              <q-icon name="trending_down" size="18px" />
            </div>
            <div class="stat-body">
              <span class="stat-label">支出总额</span>
              <span class="stat-value stat-value--expense">
                -¥{{ formatYuan(transactionStore.totalExpense) }}
              </span>
            </div>
          </div>

          <div class="stat-card stat-card--net">
            <div class="stat-icon">
              <q-icon name="swap_vert" size="18px" />
            </div>
            <div class="stat-body">
              <span class="stat-label">净流向</span>
              <span
                class="stat-value"
                :class="netFlow >= 0 ? 'stat-value--income' : 'stat-value--expense'"
              >
                {{ netFlow >= 0 ? '+' : '' }}¥{{ formatYuan(Math.abs(netFlow)) }}
              </span>
            </div>
          </div>
        </div>

        <!-- Filter Bar -->
        <div class="filter-bar">
          <div class="filter-group">
            <q-select
              v-model="filterType"
              :options="typeOptions"
              label="类型"
              dense
              outlined
              emit-value
              map-options
              class="filter-select"
            />
            <q-btn
              flat
              dense
              icon="filter_list"
              label="重置"
              class="reset-btn"
              @click="resetFilters"
            />
          </div>
        </div>

        <!-- Transaction List -->
        <div class="tx-list">
          <!-- Loading skeleton -->
          <template v-if="transactionStore.loading && transactionStore.transactions.length === 0">
            <div v-for="i in 6" :key="i" class="tx-skeleton">
              <q-skeleton type="rect" width="36px" height="36px" />
              <div class="skeleton-body">
                <q-skeleton type="text" width="120px" />
                <q-skeleton type="text" width="80px" />
              </div>
              <q-skeleton type="text" width="90px" class="skeleton-amount" />
            </div>
          </template>

          <!-- Empty state -->
          <div
            v-else-if="!transactionStore.loading && groupedTransactions.length === 0"
            class="empty-state"
          >
            <div class="empty-icon">
              <q-icon name="receipt_long" size="56px" />
            </div>
            <p class="empty-title">暂无交易记录</p>
            <p class="empty-sub">您的资金流水将显示在这里</p>
          </div>

          <!-- Grouped by date -->
          <template v-else>
            <div
              v-for="group in groupedTransactions"
              :key="group.date"
              class="tx-group"
            >
              <div class="tx-date-header">
                <span class="date-label">{{ group.dateLabel }}</span>
                <span class="date-dot" />
              </div>

              <div class="tx-rows">
                <div
                  v-for="tx in group.items"
                  :key="tx.id"
                  class="tx-row card-reveal"
                  :style="{ animationDelay: `${groupedTransactions.indexOf(group) * 40}ms` }"
                >
                  <div
                    class="tx-icon"
                    :class="tx.iconClass"
                  >
                    <q-icon
                      :name="tx.iconName"
                      size="16px"
                    />
                  </div>

                  <div class="tx-body">
                    <span class="tx-type">{{ tx.displayType }}</span>
                    <span class="tx-desc">{{ tx.description || '—' }}</span>
                  </div>

                  <div class="tx-meta">
                    <span
                      class="tx-amount"
                      :class="tx.amountClass"
                    >
                      {{ tx.amountPrefix }}¥{{ formatYuan(tx.amount) }}
                    </span>
                    <span class="tx-time">{{ formatTime(tx.createdAt) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="pagination-wrap">
          <q-pagination
            v-model="currentPage"
            :max="totalPages"
            :max-pages="6"
            color="grey-7"
            active-color="amber-8"
            boundary-numbers
            @update:model-value="onPageChange"
          />
        </div>
      </div>
    </q-pull-to-refresh>
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useTransactionStore } from 'src/stores/transaction'
import { useWalletStore } from 'src/stores/wallet'

const route = useRoute()
const transactionStore = useTransactionStore()
const walletStore = useWalletStore()

const filterType = ref('all')
const currentPage = ref(1)
const pageSize = 10
const totalPages = computed(() => Math.ceil(transactionStore.totalCount / pageSize) || 1)

// Type options match actual displayType values
const typeOptions = [
  { label: '全部', value: 'all' },
  { label: '转入', value: '转入' },
  { label: '转出', value: '转出' },
  { label: '冻结', value: '冻结' },
  { label: '解冻', value: '解冻' },
  { label: '提现解冻', value: '提现解冻' },
  { label: '充值', value: '充值' },
  { label: '提现', value: '提现' },
  { label: '悬赏收入', value: '悬赏收入' },
  { label: '悬赏支出', value: '悬赏支出' },
]

const netFlow = computed(() => transactionStore.totalIncome - transactionStore.totalExpense)

const filteredTransactions = computed(() => {
  if (filterType.value === 'all') return transactionStore.transactions
  return transactionStore.transactions.filter(tx => tx.displayType === filterType.value)
})

// Enrich each transaction with display helpers
// Skip ledger-sourced FROZEN entries (transfer already shows the frozen record)
const displayTransactions = computed(() =>
  filteredTransactions.value
    .filter(tx => {
      if (tx.source === 'ledger' && tx.displayType === '冻结') return false
      return true
    })
    .map(tx => {
      let iconName = 'arrow_forward'
      let iconClass = 'tx-icon--income'
      let amountPrefix = '+'
      let amountClass = 'tx-amount--income'

      switch (tx.direction) {
        case 'income':
          iconName = 'arrow_upward'
          iconClass = 'tx-icon--income'
          amountPrefix = '+'
          amountClass = 'tx-amount--income'
          break
        case 'expense':
          iconName = 'arrow_downward'
          iconClass = 'tx-icon--expense'
          amountPrefix = '-'
          amountClass = 'tx-amount--expense'
          break
        case 'frozen':
          iconName = 'ac_unit'
          iconClass = 'tx-icon--frozen'
          // 解冻类型不显示正负号前缀
          amountPrefix = ''
          amountClass = 'tx-amount--frozen'
          break
      }

      return { ...tx, iconName, iconClass, amountPrefix, amountClass }
    })
)

const groupedTransactions = computed(() => {
  const groups = {}
  for (const tx of displayTransactions.value) {
    const dateKey = tx.createdAt ? tx.createdAt.split('T')[0] : 'unknown'
    if (!groups[dateKey]) {
      groups[dateKey] = {
        date: dateKey,
        dateLabel: formatDateLabel(dateKey),
        items: [],
      }
    }
    groups[dateKey].items.push(tx)
  }
  return Object.values(groups).sort((a, b) => b.date.localeCompare(a.date))
})

function formatDateLabel(dateKey) {
  if (!dateKey || dateKey === 'unknown') return '其他'
  const d = new Date(dateKey + 'T00:00:00')
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  if (dateKey === today.toISOString().split('T')[0]) return '今天'
  if (dateKey === yesterday.toISOString().split('T')[0]) return '昨天'

  return d.toLocaleDateString('zh-CN', {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  })
}

function formatYuan(cents) {
  return (Number(cents) / 100).toLocaleString('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

function formatTime(isoString) {
  if (!isoString) return '—'
  const d = new Date(isoString)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadTransactions() {
  const accountId = walletStore.account?.id || route.query.account_id
  if (!accountId) {
    await walletStore.fetchAccount()
    if (walletStore.account?.id) {
      await transactionStore.fetchAll(walletStore.account.id)
    }
  } else {
    await transactionStore.fetchAll(accountId)
  }
}

async function onRefresh(done) {
  transactionStore.reset()
  currentPage.value = 1
  await loadTransactions()
  done()
}

async function onPageChange(page) {
  transactionStore.setPage(page)
  await transactionStore.fetchAll(route.query.account_id || walletStore.account?.id)
}

function resetFilters() {
  filterType.value = 'all'
}

onMounted(loadTransactions)
</script>

<style scoped lang="scss">
.history-page {
  background: var(--color-bg-primary);
  min-height: 100vh;
}

.page-inner {
  max-width: 860px;
  margin: 0 auto;
  padding: 40px 24px 80px;
}

// Header
.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 32px;
}

.back-btn {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--color-text-muted);
  transition: all 0.15s;

  &:hover {
    border-color: var(--color-accent-gold);
    color: var(--color-accent-gold);
  }
}

.page-title {
  font-family: var(--font-display);
  font-size: 1.8rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}

// Summary Stats
.summary-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 24px;

  @media (max-width: 480px) {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;

  &--income { border-color: rgba(52, 211, 153, 0.3); }
  &--expense { border-color: rgba(248, 113, 113, 0.3); }
  &--net { border-color: rgba(201, 168, 76, 0.3); }
}

.stat-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  .stat-card--income & {
    background: var(--color-glow-teal);
    color: var(--color-accent-teal);
  }
  .stat-card--expense & {
    background: rgba(248, 113, 113, 0.15);
    color: var(--color-accent-red);
  }
  .stat-card--net & {
    background: var(--color-glow-gold);
    color: var(--color-accent-gold);
  }
}

.stat-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.stat-label {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
}

.stat-value {
  font-family: var(--font-mono);
  font-size: 1.1rem;
  font-weight: 700;
  letter-spacing: -0.02em;

  &--income { color: var(--color-accent-teal); }
  &--expense { color: var(--color-accent-red); }
}

// Filter Bar
.filter-bar {
  margin-bottom: 20px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-select {
  flex: 1;
  max-width: 200px;

  :deep(.q-field__control) {
    background: var(--color-bg-elevated) !important;
    border-color: var(--color-border) !important;
  }

  :deep(.q-field__native) {
    color: #ffffff !important;
    -webkit-text-fill-color: #ffffff !important;
  }

  :deep(.q-field__label) {
    color: var(--color-text-muted) !important;
  }

  :deep(.q-field--focused .q-field__control) {
    border-color: var(--color-accent-gold) !important;
  }

  :deep(.q-field__marginal) {
    background: var(--color-bg-elevated) !important;
  }

  :deep(.q-field__append) {
    color: var(--color-text-muted) !important;
  }
}

.reset-btn {
  color: var(--color-text-muted);
  font-size: 0.8rem;
  &:hover { color: var(--color-accent-gold); }
}

// Transaction List
.tx-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tx-skeleton {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  margin-bottom: 4px;
}

.skeleton-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.skeleton-amount {
  margin-left: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px 24px;
  text-align: center;
}

.empty-icon {
  color: var(--color-text-muted);
  opacity: 0.4;
  margin-bottom: 16px;
}

.empty-title {
  font-family: var(--font-display);
  font-size: 1.2rem;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}

.empty-sub {
  font-size: 0.85rem;
  color: var(--color-text-muted);
}

// Transaction Group
.tx-group {
  margin-bottom: 8px;
}

.tx-date-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  margin-bottom: 4px;
}

.date-label {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted);
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.date-dot {
  flex: 1;
  height: 1px;
  background: var(--color-border);
  opacity: 0.5;
}

.tx-rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tx-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  transition: border-color 0.15s, transform 0.15s;

  &:hover {
    border-color: var(--color-text-muted);
    transform: translateX(4px);
  }
}

.tx-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &--income {
    background: var(--color-glow-teal);
    color: var(--color-accent-teal);
  }

  &--expense {
    background: rgba(248, 113, 113, 0.15);
    color: var(--color-accent-red);
  }

  &--frozen {
    background: rgba(251, 191, 36, 0.15);
    color: var(--color-accent-amber);
  }
}

.tx-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.tx-type {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--color-text-primary);
}

.tx-desc {
  font-size: 0.78rem;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex-shrink: 0;
}

.tx-amount {
  font-family: var(--font-mono);
  font-size: 0.95rem;
  font-weight: 700;
  letter-spacing: -0.02em;

  &--income { color: var(--color-accent-teal); }
  &--expense { color: var(--color-accent-red); }
  &--frozen { color: var(--color-accent-amber); }
}

.tx-time {
  font-family: var(--font-mono);
  font-size: 0.72rem;
  color: var(--color-text-muted);
}

// Pagination
.pagination-wrap {
  display: flex;
  justify-content: center;
  padding: 24px 0;

  :deep(.q-pagination__middle) {
    color: var(--color-text-muted);
  }
}
</style>
