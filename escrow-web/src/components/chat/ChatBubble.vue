<template>
  <div class="chat-bubble" :class="{ own: isOwn }">
    <div class="bubble-content" :data-sender="message.sender_username">
      <div class="bubble-text">{{ message.content }}</div>
      <div class="bubble-time">{{ formatTime(message.created_at) }}</div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  message: { type: Object, required: true },
  isOwn: { type: Boolean, default: false },
})

function formatTime(ts) {
  if (!ts) return ''
  const date = new Date(ts * 1000)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()
  if (isToday) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }) +
    ' ' + date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped lang="scss">
.chat-bubble {
  display: flex;
  justify-content: flex-start;

  &.own {
    justify-content: flex-end;

    .bubble-content {
      background: var(--color-accent-gold);
      color: #1a1a1a;
      border-radius: 16px 16px 4px 16px;
    }

    .bubble-time {
      color: rgba(26, 26, 26, 0.5);
    }
  }
}

.bubble-content {
  max-width: 75%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 16px 16px 16px 4px;
  padding: 8px 12px;
}

.bubble-text {
  font-size: 0.9rem;
  line-height: 1.4;
  word-break: break-word;
  white-space: pre-wrap;
}

.bubble-time {
  font-size: 0.7rem;
  color: var(--color-text-muted);
  text-align: right;
  margin-top: 4px;
}
</style>
