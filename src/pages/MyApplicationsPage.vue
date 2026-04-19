<template>
  <q-page class="my-apps-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">我的接单</h1>
        <p class="page-subtitle">您申请过的悬赏任务</p>
      </div>

      <div v-if="bountyStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <div v-else-if="myApplications.length" class="app-list">
        <q-card
          v-for="(item, i) in myApplications"
          :key="item.id"
          class="app-card card-reveal"
          :style="{ animationDelay: `${i * 80}ms` }"
          @click="$router.push(`/bounty/${item.bounty.id}`)"
        >
          <div :class="['status-bar', statusBarClass(item.bounty.status)]" />
          <q-card-section class="app-content">
            <div class="app-header">
              <StatusBadge :status="item.bounty.status" />
              <AmountDisplay :cents="item.bounty.reward_amount" />
            </div>
            <h3 class="app-title">{{ item.bounty.title }}</h3>
            <div class="app-meta">
              <span>申请于 {{ formatDate(item.created_at) }}</span>
              <StatusBadge :status="item.status" />
            </div>
          </q-card-section>
        </q-card>
      </div>

      <div v-else class="empty-state">
        <div class="empty-icon">◈</div>
        <h3>暂无接单</h3>
        <p>您还没有申请过任何悬赏任务</p>
        <q-btn unelevated color="primary" label="浏览悬赏" to="/" class="q-mt-md" />
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useBountyStore } from 'src/stores/bounty'
import { useAuthStore } from 'src/stores/auth'
import StatusBadge from 'src/components/bounty/StatusBadge.vue'
import AmountDisplay from 'src/components/bounty/AmountDisplay.vue'

const bountyStore = useBountyStore()
const authStore = useAuthStore()

const myApplications = computed(() =>
  bountyStore.bounties
    .flatMap(b => (b.applications || [])
      .filter(a => a.hunter_username === authStore.username)
      .map(a => ({ ...a, bounty: b }))
    )
)

function statusBarClass(status) {
  const map = {
    PAYING: 'bar-paying', PENDING: 'bar-pending', IN_PROGRESS: 'bar-progress',
    COMPLETED: 'bar-completed', FAILED: 'bar-failed', CANCELED: 'bar-canceled',
  }
  return map[status] ?? 'bar-pending'
}

function formatDate(d) {
  if (!d) return '—'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('zh-CN')
}

onMounted(() => bountyStore.fetchBounties({ pageSize: 50 }))
</script>

<style scoped lang="scss">
.my-apps-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 860px; margin: 0 auto; padding: 48px 24px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; margin-bottom: 6px; }
.page-subtitle { color: var(--color-text-muted); font-size: 0.9rem; margin-bottom: 32px; }
.loading-area { display: flex; justify-content: center; padding: 80px; }

.app-list { display: flex; flex-direction: column; gap: 16px; }

.app-card {
  position: relative;
  overflow: hidden;
  cursor: pointer;
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  &:hover { transform: translateX(4px); border-color: var(--color-accent-teal) !important; }
}

.status-bar { height: 3px; }
.bar-paying    { background: var(--color-accent-gold); }
.bar-pending  { background: var(--color-accent-teal); }
.bar-progress { background: var(--color-accent-amber); }
.bar-completed { background: var(--color-accent-green); }
.bar-failed, .bar-canceled  { background: var(--color-accent-red); }

.app-content { padding: 20px; }
.app-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.app-title { font-family: var(--font-display); font-size: 1.1rem; font-weight: 600; margin-bottom: 10px; }
.app-meta { display: flex; justify-content: space-between; align-items: center; font-size: 0.78rem; color: var(--color-text-muted); font-family: var(--font-mono); }

.empty-state {
  text-align: center; padding: 80px 24px;
  .empty-icon { font-size: 4rem; color: var(--color-border); margin-bottom: 16px; }
  h3 { font-family: var(--font-display); font-size: 1.5rem; color: var(--color-text-muted); margin-bottom: 8px; }
  p { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }
}
</style>
