<template>
  <div class="comment-item">
    <div class="comment-header">
      <img
        v-if="comment.authorAvatarUrl"
        :src="imageUrl(comment.authorAvatarUrl)"
        class="comment-avatar"
        alt="avatar"
        @click="$router.push(`/profile/${comment.authorUsername}`)"
      />
      <div
        v-else
        class="comment-avatar"
        @click="$router.push(`/profile/${comment.authorUsername}`)"
      >
        {{ comment.authorUsername?.[0]?.toUpperCase() || '?' }}
      </div>
      <span class="comment-author" @click="$router.push(`/profile/${comment.authorUsername}`)">
        {{ comment.authorUsername }}
      </span>
      <span class="comment-time">{{ formatDate(comment.created_at) }}</span>
      <q-btn
        flat dense size="sm" icon="reply" label="回复"
        @click="$emit('reply', comment)"
      />
      <q-btn
        v-if="isOwn"
        flat dense size="sm" icon="delete" color="negative"
        @click="$emit('delete', comment.id)"
      />
    </div>
    <!-- 根评论本身不存在 reply_to_id，所以这里不显示回复引用 -->
    <div v-if="replyToComment" class="comment-quote">
      <span>@{{ replyToComment.authorUsername }}: {{ replyToComment.content }}</span>
    </div>
    <p class="comment-content">{{ comment.content }}</p>

    <q-img
      v-if="comment.imagePath"
      :src="imageUrl(comment.imagePath)"
      class="comment-image"
      style="max-width: 200px; cursor: pointer; margin-left: 36px; margin-bottom: 8px;"
      @click="previewImage(imageUrl(comment.imagePath))"
    />

    <!-- 子评论（两级楼中楼：全部扁平展示，无递归） -->
    <div v-if="comment.replies?.length" class="comment-replies">
      <ReplyItem
        v-for="reply in comment.replies"
        :key="reply.id"
        :reply="reply"
        :all-comments="allComments"
        @reply="$emit('reply', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useQuasar } from 'quasar'
import { useAuthStore } from 'src/stores/auth'
import { imageUrl } from 'src/api/upload'
import ReplyItem from './ReplyItem.vue'

const props = defineProps({
  comment: { type: Object, required: true },
  allComments: { type: Array, default: () => [] },
})
defineEmits(['reply', 'delete'])

const authStore = useAuthStore()
const $q = useQuasar()
const isOwn = computed(() => props.comment.authorUsername === authStore.username)

const replyToComment = computed(() => {
  if (!props.comment.replyToId) return null
  const id = String(props.comment.replyToId)
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
.comment-item {
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border);
  &:last-child { border-bottom: none; }
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}

.comment-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  cursor: pointer;
  flex-shrink: 0;
  object-fit: cover;
}

img.comment-avatar {
  display: block;
}

.comment-author {
  font-size: 0.88rem;
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
  margin: 0 0 8px 36px;
  font-size: 0.9rem;
  color: var(--color-text-primary);
  line-height: 1.5;
  word-break: break-word;
}

.comment-quote {
  margin: 0 0 4px 36px;
  padding: 4px 10px;
  background: rgba(201, 168, 76, 0.15);
  border-left: 2px solid var(--color-accent-gold);
  border-radius: 4px;
  font-size: 0.8rem;
  color: var(--color-accent-gold);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  box-sizing: border-box;

  span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.comment-replies {
  margin-left: 20px;
}
</style>
