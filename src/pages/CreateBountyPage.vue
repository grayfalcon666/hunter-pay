<template>
  <q-page class="create-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">发布悬赏</h1>
        <p class="page-subtitle">发布任务，吸引猎人接单</p>
      </div>

      <q-card class="create-card">
        <q-card-section>
          <q-form @submit="handleCreate">

            <q-input
              v-model="form.title"
              label="悬赏标题"
              outlined
              counter
              maxlength="100"
              :rules="[v => !!v || '请输入标题']"
              class="q-mb-lg"
            >
              <template #prepend><q-icon name="title" color="grey-6" /></template>
            </q-input>

            <q-input
              v-model="form.description"
              label="悬赏描述"
              type="textarea"
              outlined
              autogrow
              counter
              maxlength="2000"
              :rules="[v => !!v || '请输入描述']"
              class="q-mb-lg"
            >
              <template #prepend><q-icon name="description" color="grey-6" /></template>
            </q-input>

            <div class="amount-section">
              <div class="section-label">
                <q-icon name="payments" color="amber" />
                <span>悬赏金额</span>
              </div>

              <div class="amount-inputs">
                <q-input
                  v-model.number="form.amountYuan"
                  label="金额（元）"
                  outlined
                  type="number"
                  min="1"
                  suffix="元"
                  class="amount-input"
                  :rules="[v => v > 0 || '金额必须大于0']"
                />
                <div class="amount-preview">
                  <span class="preview-label">实际到账（分）</span>
                  <span class="preview-value">{{ form.amountYuan * 100 }}</span>
                </div>
              </div>

              <div class="amount-tip">
                <q-icon name="info" size="14px" />
                金额将立即从您的账户余额中扣除
              </div>
            </div>

            <q-separator color="border" class="q-my-xl" />

            <div class="deadline-section">
              <div class="section-label">
                <q-icon name="schedule" color="amber" />
                <span>截止时间</span>
              </div>
              <div class="deadline-inputs">
                <q-input
                  v-model="form.deadlineDate"
                  label="截止日期"
                  outlined
                  type="date"
                  class="deadline-input"
                  :rules="[v => !!form.deadlineDate || !!form.deadlineTime || '请填写截止时间']"
                />
                <q-input
                  v-model="form.deadlineTime"
                  label="截止时间"
                  outlined
                  type="time"
                  class="deadline-input"
                  :rules="[v => !!form.deadlineDate || !!form.deadlineTime || '请填写截止时间']"
                />
              </div>
              <div class="deadline-tip">
                <q-icon name="info" size="14px" />
                设置截止时间后，猎人需在规定时间内完成工作，超时将影响信誉分
              </div>
            </div>

            <q-separator color="border" class="q-my-xl" />

            <div class="form-actions">
              <q-btn flat label="取消" to="/" class="cancel-btn" />
              <q-btn
                type="submit"
                unelevated
                color="primary"
                label="发布悬赏"
                :loading="loading"
                :disable="!canSubmit"
                class="submit-btn"
              />
            </div>

          </q-form>
        </q-card-section>
      </q-card>
    </div>
  </q-page>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useBountyStore } from 'src/stores/bounty'

const bountyStore = useBountyStore()
const router = useRouter()
const $q = useQuasar()

const form = ref({ title: '', description: '', amountYuan: 100, deadlineDate: '', deadlineTime: '' })
const loading = ref(false)

// 截止时间是否有效（必须大于当前时间）
const isDeadlineValid = computed(() => {
  if (!form.value.deadlineDate && !form.value.deadlineTime) return false
  const now = new Date()
  const dateStr = form.value.deadlineDate || now.toISOString().split('T')[0]
  const timeStr = form.value.deadlineTime || '23:59'
  return new Date(`${dateStr}T${timeStr}:00`) > now
})

const canSubmit = computed(() =>
  form.value.title &&
  form.value.description &&
  form.value.amountYuan > 0 &&
  form.value.deadlineDate &&
  form.value.deadlineTime &&
  isDeadlineValid.value
)

async function handleCreate() {
  loading.value = true
  try {
    const amountCents = Math.round(form.value.amountYuan * 100)
    const payload = { title: form.value.title, description: form.value.description, reward_amount: amountCents }
    // Add deadline if set
    if (form.value.deadlineDate && form.value.deadlineTime) {
      const deadlineTs = Math.floor(new Date(`${form.value.deadlineDate}T${form.value.deadlineTime}:00`).getTime() / 1000)
      payload.deadline_timestamp = deadlineTs
    }
    await bountyStore.createBounty(payload)
    $q.notify({ type: 'positive', message: '悬赏发布成功' })
    router.push('/')
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '发布失败，请检查余额' })
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.create-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 720px; margin: 0 auto; padding: 48px 24px; }
.page-header { margin-bottom: 32px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; margin-bottom: 6px; }
.page-subtitle { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }

.create-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.amount-section {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  padding: 20px;
  margin-bottom: 8px;
}

.section-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 16px;
}

.amount-inputs {
  display: flex;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
}

.amount-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
  .preview-label { font-size: 0.75rem; color: var(--color-text-muted); }
  .preview-value {
    font-family: var(--font-mono);
    font-size: 1.2rem;
    font-weight: 700;
    color: var(--color-accent-gold);
  }
}

.amount-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
  color: var(--color-text-muted);
  margin-top: 12px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 8px;
}

.cancel-btn { color: var(--color-text-muted); }

.deadline-section {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  padding: 20px;
  margin-bottom: 8px;
}

.deadline-inputs {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.deadline-input { flex: 1; min-width: 160px; }

.deadline-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
  color: var(--color-text-muted);
  margin-top: 12px;
}

.submit-btn {
  min-width: 140px;
  height: 44px;
  font-size: 0.95rem;
  font-weight: 600;
}
</style>
