<template>
  <q-page class="my-comments-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">我的评论</h1>
        <p class="page-subtitle">您发表过的所有评论</p>
      </div>

      <div v-if="loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <div v-else-if="comments.length" class="comment-list">
        <q-card
          v-for="(comment, i) in comments"
          :key="comment.id"
          class="comment-card card-reveal"
          :style="{ animationDelay: `${i * 80}ms` }"
        >
          <q-card-section>
            <div class="comment-bounty">
              <router-link :to="`/bounty/${comment.bounty_id}`" class="bounty-link">
                悬赏 #{{ comment.bounty_id }}
              </router-link>
            </div>
            <p class="comment-text">{{ comment.content }}</p>
            <div class="comment-meta">
              <span>{{ formatDate(comment.created_at) }}</span>
              <q-btn
                flat dense size="sm" icon="delete" color="negative"
                @click="handleDelete(comment.id)"
              />
            </div>
          </q-card-section>
        </q-card>
      </div>

      <div v-else class="empty-state">
        <div class="empty-icon">◈</div>
        <h3>暂无评论</h3>
        <p>您还没有发表过任何评论</p>
        <q-btn unelevated color="primary" label="浏览悬赏" to="/" class="q-mt-md" />
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useAuthStore } from 'src/stores/auth'
import { listUserComments, deleteComment } from 'src/api/chat'

const $q = useQuasar()
const authStore = useAuthStore()
const comments = ref([])
const loading = ref(false)

async function loadComments() {
  loading.value = true
  try {
    const data = await listUserComments(authStore.username)
    comments.value = data.comments || []
  } catch {
    $q.notify({ type: 'negative', message: '加载评论失败' })
  } finally {
    loading.value = false
  }
}

async function handleDelete(commentId) {
  $q.dialog({
    title: '确认删除',
    message: '确定要删除这条评论吗？',
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    try {
      await deleteComment(commentId)
      comments.value = comments.value.filter(c => c.id !== commentId)
      $q.notify({ type: 'positive', message: '评论已删除' })
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '删除失败' })
    }
  })
}

function formatDate(d) {
  if (!d) return '—'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('zh-CN')
}

onMounted(loadComments)
</script>

<style scoped lang="scss">
.my-comments-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 860px; margin: 0 auto; padding: 48px 24px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; margin-bottom: 6px; }
.page-subtitle { color: var(--color-text-muted); font-size: 0.9rem; margin-bottom: 32px; }
.loading-area { display: flex; justify-content: center; padding: 80px; }

.comment-list { display: flex; flex-direction: column; gap: 16px; }

.comment-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.comment-bounty { margin-bottom: 8px; }
.bounty-link { color: var(--color-accent-teal); text-decoration: none; font-size: 0.85rem; &:hover { text-decoration: underline; } }
.comment-text { font-size: 0.95rem; color: var(--color-text-primary); margin: 0 0 10px; line-height: 1.5; }
.comment-meta { display: flex; justify-content: space-between; align-items: center; font-size: 0.78rem; color: var(--color-text-muted); font-family: var(--font-mono); }

.empty-state {
  text-align: center; padding: 80px 24px;
  .empty-icon { font-size: 4rem; color: var(--color-border); margin-bottom: 16px; }
  h3 { font-family: var(--font-display); font-size: 1.5rem; color: var(--color-text-muted); margin-bottom: 8px; }
  p { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }
}
</style>
