<template>
  <span :class="['status-badge', statusClass]">{{ label }}</span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: { type: String, required: true },
})

const config = {
  PAYING:      { label: '付款中',   cls: 'status-paying' },
  PENDING:     { label: '待接单',   cls: 'status-pending' },
  IN_PROGRESS: { label: '进行中',   cls: 'status-progress' },
  SUBMITTED:   { label: '待审核',   cls: 'status-progress' },
  COMPLETED:   { label: '已完成',   cls: 'status-completed' },
  REJECTED:    { label: '已拒绝',   cls: 'status-failed' },
  FAILED:      { label: '失败',     cls: 'status-failed' },
  CANCELED:    { label: '已取消',   cls: 'status-canceled' },
  EXPIRED:     { label: '已过期',   cls: 'status-failed' },
}

const label = computed(() => config[props.status]?.label ?? props.status)
const statusClass = computed(() => config[props.status]?.cls ?? '')
</script>

<style scoped lang="scss">
.status-badge {
  display: inline-block;
  font-family: var(--font-mono);
  font-size: 0.68rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: 4px;
  border: 1px solid;
}

.status-paying    { color: var(--color-accent-gold);  border-color: var(--color-accent-gold);  background: rgba(201,168,76,0.1); }
.status-pending   { color: var(--color-accent-teal);   border-color: var(--color-accent-teal);   background: rgba(45,212,191,0.1); }
.status-progress  { color: var(--color-accent-amber); border-color: var(--color-accent-amber); background: rgba(251,191,36,0.1); }
.status-completed { color: var(--color-accent-green); border-color: var(--color-accent-green); background: rgba(52,211,153,0.1); }
.status-failed,
.status-canceled  { color: var(--color-accent-red);   border-color: var(--color-accent-red);   background: rgba(248,113,113,0.1); }
</style>
