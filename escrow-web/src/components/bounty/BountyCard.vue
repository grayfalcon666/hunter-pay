<template>
  <q-card class="bounty-card" @click="$router.push(`/bounty/${bounty.id}`)">
    <!-- 状态色条 -->
    <div :class="['status-bar', statusBarClass]" />

    <q-card-section class="card-content">
      <!-- 头部：状态 + 金额 -->
      <div class="card-header">
        <StatusBadge :status="bounty.status" />
        <AmountDisplay :cents="bounty.rewardAmount ?? bounty.reward_amount" class="card-amount" />
      </div>

      <!-- 标题 -->
      <h3 class="card-title">{{ bounty.title }}</h3>

      <!-- 描述 -->
      <p class="card-desc">{{ bounty.description }}</p>

      <q-separator color="border" class="q-my-md" />

      <!-- 底部信息 -->
      <div class="card-footer">
        <div class="employer-info">
          <img v-if="bounty.employerAvatarUrl" :src="imageUrl(bounty.employerAvatarUrl)" class="employer-avatar" alt="avatar" />
          <div v-else class="employer-avatar">{{ bounty.employerUsername?.[0]?.toUpperCase() ?? '?' }}</div>
          <span class="employer-name">{{ bounty.employerUsername ?? bounty.employer_username ?? '未知' }}</span>
        </div>
        <div class="card-meta">
          <span v-if="bounty.deadline" class="card-deadline">
            <q-icon name="schedule" size="12px" />
            {{ formatDeadline }}
          </span>
          <span class="card-time">{{ relativeTime }}</span>
        </div>
      </div>
    </q-card-section>
  </q-card>
</template>

<script setup>
import { computed } from 'vue'
import StatusBadge from './StatusBadge.vue'
import AmountDisplay from './AmountDisplay.vue'
import { imageUrl } from 'src/api/upload'

const props = defineProps({
  bounty: { type: Object, required: true },
})

const statusBarClass = computed(() => {
  const map = {
    PAYING: 'bar-paying', PENDING: 'bar-pending', IN_PROGRESS: 'bar-progress',
    COMPLETED: 'bar-completed', FAILED: 'bar-failed', CANCELED: 'bar-canceled',
  }
  return map[props.bounty.status] ?? 'bar-pending'
})

const relativeTime = computed(() => {
  const dateStr = props.bounty.createdAt || props.bounty.created_at
  if (!dateStr) return '未知'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now - date) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)}天前`
  return date.toLocaleDateString('zh-CN')
})

const formatDeadline = computed(() => {
  const d = props.bounty.deadline
  if (!d) return ''
  const date = new Date(d)
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
})
</script>

<style scoped lang="scss">
.bounty-card {
  position: relative;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  flex-direction: column;

  &:hover {
    border-color: var(--color-accent-gold) !important;
    box-shadow: 0 8px 40px rgba(0,0,0,0.5), 0 0 0 1px var(--color-glow-gold);
  }
}

.status-bar {
  height: 3px;
  flex-shrink: 0;
}
.bar-paying    { background: var(--color-accent-gold); }
.bar-pending  { background: var(--color-accent-teal); }
.bar-progress { background: var(--color-accent-amber); }
.bar-submitted { background: var(--color-accent-amber); }
.bar-rejected { background: var(--color-accent-red); }
.bar-expired { background: var(--color-accent-red); }
.bar-completed { background: var(--color-accent-green); }
.bar-failed,
.bar-canceled  { background: var(--color-accent-red); }

.card-content {
  display: flex;
  flex-direction: column;
  flex: 1;
  padding: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-amount {
  font-size: 1.1rem;
}

.card-title {
  font-family: var(--font-display);
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--color-text-primary);
  line-height: 1.3;
  margin-bottom: 8px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-desc {
  font-size: 0.85rem;
  color: var(--color-text-muted);
  line-height: 1.5;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  margin: 0;
  flex: 1;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-deadline {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 0.75rem;
  color: var(--color-accent-amber);
  font-family: var(--font-mono);
}

.employer-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.employer-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  object-fit: cover;
}

img.employer-avatar {
  display: block;
}

.employer-name {
  font-size: 0.85rem;
  color: var(--color-text-muted);
}

.card-time {
  font-size: 0.78rem;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
}
</style>
