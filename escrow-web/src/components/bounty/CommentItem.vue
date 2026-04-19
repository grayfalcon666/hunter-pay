<template>
  <div class="comment-item" :class="{ 'is-reply': depth > 0 }">
    <div class="comment-header">
      <div class="comment-avatar" @click="$router.push(`/profile/${comment.author_username}`)">
        {{ comment.author_username?.[0]?.toUpperCase() || '?' }}
      </div>
      <span class="comment-author" @click="$router.push(`/profile/${comment.author_username}`)">
        {{ comment.author_username }}
      </span>
      <span class="comment-time">{{ formatDate(comment.created_at) }}</span>
      <q-btn
        flat dense size="sm" icon="reply" label="回复"
        @click="$emit('reply', comment)"
        v-if="depth < 3"
      />
      <q-btn
        v-if="isOwn"
        flat dense size="sm" icon="delete" color="negative"
        @click="$emit('delete', comment.id)"
      />
    </div>
    <div v-if="comment.parent_author_username" class="reply-quote">
      回复 @{{ comment.parent_author_username }}
    </div>
    <p class="comment-content">{{ comment.content }}</p>
    <!-- Nested replies (max depth 3) -->
    <div v-if="comment.replies?.length && depth < 3" class="comment-replies">
      <CommentItem
        v-for="reply in comment.replies"
        :key="reply.id"
        :comment="reply"
        :depth="depth + 1"
        @reply="$emit('reply', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from 'src/stores/auth'

const props = defineProps({
  comment: { type: Object, required: true },
  depth: { type: Number, default: 0 },
})
defineEmits(['reply', 'delete'])

const authStore = useAuthStore()
const isOwn = computed(() => props.comment.author_username === authStore.username)

function formatDate(ts) {
  if (!ts) return ''
  const date = new Date(ts * 1000)
  const now = new Date()
  const diff = now - date
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)} 天前`
  return date.toLocaleDateString('zh-CN')
}
</script>

<style scoped lang="scss">
.comment-item {
  &.is-reply {
    margin-left: 20px;
    padding-left: 12px;
    border-left: 2px solid var(--color-border);
  }
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}

.comment-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  cursor: pointer;
  flex-shrink: 0;
}

.comment-author {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-primary);
  cursor: pointer;
  &:hover { color: var(--color-accent-gold); }
}

.comment-time {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.comment-content {
  margin: 0 0 8px 32px;
  font-size: 0.9rem;
  color: var(--color-text-primary);
  line-height: 1.5;
  word-break: break-word;
}

.reply-quote {
  margin: 0 0 6px 32px;
  font-size: 0.82rem;
  color: var(--color-accent-teal);
  font-style: italic;
}

.comment-replies {
  margin-top: 8px;
}
</style>
