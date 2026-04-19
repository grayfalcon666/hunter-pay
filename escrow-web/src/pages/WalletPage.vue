<template>
  <q-page class="wallet-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">我的钱包</h1>
      </div>

      <!-- 余额卡片 -->
      <q-card class="balance-card">
        <div class="balance-glow" />
        <q-card-section class="balance-content">
          <span class="balance-label">账户余额</span>
          <div v-if="walletStore.loading" class="balance-loading">
            <q-skeleton type="text" width="200px" />
          </div>
          <div v-else class="balance-value">
            <span class="currency-symbol">¥</span>
            <span class="balance-amount">{{ formatYuan(walletStore.account?.balance || 0) }}</span>
          </div>
          <span class="balance-hint">账户 ID: <span class="mono">#{{ walletStore.account?.id || '—' }}</span></span>
        </q-card-section>
      </q-card>

      <div class="wallet-actions">
        <!-- 充值 -->
        <q-card class="action-card">
          <q-card-section>
            <div class="action-icon-wrap topup-icon">
              <q-icon name="add_circle" size="28px" />
            </div>
            <h3 class="action-title">充值</h3>
            <p class="action-desc">通过支付宝充值到账户余额</p>
            <q-btn unelevated color="primary" label="立即充值" @click="showTopUp = true" />
          </q-card-section>
        </q-card>

        <!-- 提现 -->
        <q-card class="action-card">
          <q-card-section>
            <div class="action-icon-wrap withdraw-icon">
              <q-icon name="account_balance" size="28px" />
            </div>
            <h3 class="action-title">提现</h3>
            <p class="action-desc">将余额提现到支付宝账户</p>
            <q-btn unelevated color="teal" label="申请提现" @click="showWithdraw = true" />
          </q-card-section>
        </q-card>

        <!-- 收支明细 -->
        <q-card class="action-card" @click="router.push('/wallet/history')" style="cursor:pointer">
          <q-card-section>
            <div class="action-icon-wrap history-icon">
              <q-icon name="receipt_long" size="28px" />
            </div>
            <h3 class="action-title">收支明细</h3>
            <p class="action-desc">查看账户资金流水记录</p>
            <q-btn flat color="grey-5" label="查看全部" />
          </q-card-section>
        </q-card>
      </div>

      <!-- 充值对话框 -->
      <q-dialog v-model="showTopUp">
        <q-card class="dialog-card">
          <q-card-section>
            <h3 class="dialog-title">充值</h3>
          </q-card-section>
          <q-card-section>
            <q-input
              v-model.number="topUpAmount"
              label="充值金额（元）"
              type="number"
              min="1"
              outlined
              suffix="元"
              class="q-mb-md"
            />
            <div class="quick-amounts">
              <q-btn
                v-for="amt in [50, 100, 500, 1000]"
                :key="amt"
                flat
                dense
                :label="`${amt}元`"
                @click="topUpAmount = amt"
                :class="['quick-btn', { active: topUpAmount === amt }]"
              />
            </div>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat label="取消" v-close-popup />
            <q-btn
              unelevated
              color="primary"
              label="跳转支付宝"
              :loading="topUpLoading"
              @click="handleTopUp"
            />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 提现对话框 -->
      <q-dialog v-model="showWithdraw">
        <q-card class="dialog-card">
          <q-card-section>
            <h3 class="dialog-title">提现申请</h3>
          </q-card-section>
          <q-card-section>
            <q-input
              v-model="withdrawForm.alipay_account"
              label="支付宝账号"
              outlined
              class="q-mb-md"
            />
            <q-input
              v-model="withdrawForm.alipay_real_name"
              label="真实姓名"
              outlined
              class="q-mb-md"
            />
            <q-input
              v-model.number="withdrawForm.amount"
              label="提现金额（元）"
              type="number"
              min="1"
              outlined
              suffix="元"
            />
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat label="取消" v-close-popup />
            <q-btn
              unelevated
              color="teal"
              label="提交申请"
              :loading="withdrawLoading"
              @click="handleWithdraw"
            />
          </q-card-actions>
        </q-card>
      </q-dialog>

    </div>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useWalletStore } from 'src/stores/wallet'
import { useRouter } from 'vue-router'

const walletStore = useWalletStore()
const $q = useQuasar()
const router = useRouter()

const showTopUp = ref(false)
const showWithdraw = ref(false)
const topUpAmount = ref(100)
const topUpLoading = ref(false)
const withdrawLoading = ref(false)
const withdrawForm = ref({ alipay_account: '', alipay_real_name: '', amount: 0 })

function formatYuan(cents) {
  return (cents / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function handleTopUp() {
  if (!topUpAmount.value || topUpAmount.value <= 0) {
    $q.notify({ type: 'warning', message: '请输入有效金额' })
    return
  }
  topUpLoading.value = true
  try {
    const amountCents = Math.round(topUpAmount.value * 100)
    await walletStore.topUp(walletStore.account?.id, amountCents)
    showTopUp.value = false
    $q.notify({ type: 'info', message: '充值请求已提交，请完成支付宝付款' })
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '充值失败' })
  } finally {
    topUpLoading.value = false
  }
}

async function handleWithdraw() {
  if (!withdrawForm.value.alipay_account || !withdrawForm.value.amount) {
    $q.notify({ type: 'warning', message: '请填写完整信息' })
    return
  }
  withdrawLoading.value = true
  try {
    const amountCents = Math.round(withdrawForm.value.amount * 100)
    await walletStore.withdraw({
      account_id: walletStore.account?.id,
      alipay_account: withdrawForm.value.alipay_account,
      alipay_real_name: withdrawForm.value.alipay_real_name,
      amount: amountCents,
    })
    showWithdraw.value = false
    $q.notify({ type: 'positive', message: '提现申请已提交' })
    await walletStore.fetchAccount()
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '提现失败' })
  } finally {
    withdrawLoading.value = false
  }
}

onMounted(() => walletStore.fetchAccount())
</script>

<style scoped lang="scss">
.wallet-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 860px; margin: 0 auto; padding: 48px 24px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; margin-bottom: 32px; }

.balance-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
  margin-bottom: 32px;
  position: relative;
  overflow: hidden;
}

.balance-glow {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  width: 400px;
  height: 200px;
  background: radial-gradient(ellipse at center, rgba(201,168,76,0.12) 0%, transparent 70%);
  pointer-events: none;
}

.balance-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px;
}

.balance-label {
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
  margin-bottom: 12px;
}

.balance-value {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-bottom: 12px;
}

.currency-symbol {
  font-family: var(--font-mono);
  font-size: 1.5rem;
  color: var(--color-accent-gold);
}

.balance-amount {
  font-family: var(--font-mono);
  font-size: 3rem;
  font-weight: 700;
  color: var(--color-accent-gold);
  letter-spacing: -0.03em;
}

.balance-hint { font-size: 0.8rem; color: var(--color-text-muted); }
.mono { font-family: var(--font-mono); color: var(--color-text-primary); }

.wallet-actions {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 20px;
  @media (max-width: 700px) { grid-template-columns: 1fr; }
}

.action-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
  text-align: center;
  padding: 8px;
  &:hover { border-color: var(--color-border); }
}

.action-icon-wrap {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  &.topup-icon { background: var(--color-glow-gold); color: var(--color-accent-gold); }
  &.withdraw-icon { background: var(--color-glow-teal); color: var(--color-accent-teal); }
  &.history-icon { background: var(--color-glow-gold); color: var(--color-accent-gold); }
}

.action-title { font-family: var(--font-display); font-size: 1.1rem; font-weight: 600; margin-bottom: 6px; }
.action-desc { font-size: 0.8rem; color: var(--color-text-muted); margin-bottom: 16px; }

.dialog-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  min-width: 380px;
}

.dialog-title { font-family: var(--font-display); font-size: 1.2rem; font-weight: 600; }

.quick-amounts {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.quick-btn {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--color-text-muted);
  &.active {
    border-color: var(--color-accent-gold);
    color: var(--color-accent-gold);
    background: var(--color-glow-gold);
  }
}
</style>
