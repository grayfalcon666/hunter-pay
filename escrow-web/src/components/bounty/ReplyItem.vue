<template>
  <div class="reply-item">
    <div class="reply-header">
      <img
        v-if="reply.authorAvatarUrl"
        :src="imageUrl(reply.authorAvatarUrl)"
        class="reply-avatar"
        alt="avatar"
        @click="$router.push(`/profile/${reply.authorUsername}`)"
      />
      <div
        v-else
        class="reply-avatar"
        @click="$router.push(`/profile/${reply.authorUsername}`)"
      >
        {{ reply.authorUsername?.[0]?.toUpperCase() || '?' }}
      </div>
      <span class="reply-author" @click="$router.push(`/profile/${reply.authorUsername}`)">
        {{ reply.authorUsername }}
      </span>
      <span v-if="reply.replyToUsername" class="reply-arrow">→</span>
      <span v-if="reply.replyToUsername" class="reply-to">@{{ reply.replyToUsername }}</span>
      <span class="reply-time">{{ formatDate(reply.created_at) }}</span>
      <q-btn
        flat dense size="sm" icon="reply" label="回复"
        @click="$emit('reply', reply)"
      />
      <q-btn
        v-if="isOwn"
        flat dense size="sm" icon="delete" color="negative"
        @click="$emit('delete', reply.id)"
      />
    </div>
    <div v-if="replyToComment" class="comment-quote">
      <span>@{{ replyToComment.authorUsername }}: {{ replyToComment.content }}</span>
    </div>
    <p class="reply-content">{{ reply.content }}</p>

    <q-img
      v-if="reply.imagePath"
      :src="imageUrl(reply.imagePath)"
      class="reply-image"
      style="max-width: 160px; cursor: pointer; margin-left: 26px; margin-bottom: 8px;"
      @click="previewImage(imageUrl(reply.imagePath))"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { useAuthStore } from 'src/stores/auth'
import { imageUrl } from 'src/api/upload'

const props = defineProps({
  reply: { type: Object, required: true },
  allComments: { type: Array, default: () => [] },
})
defineEmits(['reply', 'delete'])

const authStore = useAuthStore()
const $q = useQuasar()
const isOwn = computed(() => props.reply.authorUsername === authStore.username)

const replyToComment = computed(() => {
  if (!props.reply.replyToId) return null
  const id = String(props.reply.replyToId)
  return props.allComments.find(c => String(c.id) === id) || null
})

function previewImage(src) {
  $q.dialog({ title: '图片预览', html: true, message: `<img src="${src}" style="max-width:100%;max-height:70vh;" />` })
}

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
.reply-item {
  padding: 8px 0 8px 16px;
  border-left: 2px solid var(--color-border);
  margin-top: 8px;
}

.reply-header {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 4px;
}

.reply-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  cursor: pointer;
  flex-shrink: 0;
  object-fit: cover;
}

img.reply-avatar {
  display: block;
}

.reply-author {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--color-text-primary);
  cursor: pointer;
  &:hover { color: var(--color-accent-gold); }
}

.reply-arrow {
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.reply-to {
  font-size: 0.78rem;
  color: var(--color-accent-teal);
}

.reply-time {
  font-size: 0.72rem;
  color: var(--color-text-muted);
}

.reply-content {
  margin: 0 0 0 26px;
  font-size: 0.88rem;
  color: var(--color-text-primary);
  line-height: 1.5;
  word-break: break-word;
}

.comment-quote {
  margin: 0 0 4px 26px;
  background: rgba(201, 168, 76, 0.15);
  border-left: 2px solid var(--color-accent-gold);
  color: var(--color-accent-gold);
}
</style>
