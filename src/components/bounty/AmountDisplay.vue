<template>
  <span class="amount-display">{{ formatted }}</span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  // 单位：分（cents）
  cents: { type: Number, required: true },
  size: { type: String, default: 'md' }, // sm | md | lg
})

const formatted = computed(() => {
  const yuan = props.cents / 100
  return `¥ ${yuan.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
})
</script>

<style scoped lang="scss">
.amount-display {
  font-family: var(--font-mono);
  font-weight: 700;
  color: var(--color-accent-gold);
  letter-spacing: -0.02em;
}
</style>
