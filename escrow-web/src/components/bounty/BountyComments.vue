<template>
  <div class="comments-section">
    <h3 class="comments-title">评论区 ({{ totalCount }})</h3>

    <!-- New comment form -->
    <div class="comment-form">
      <div v-if="replyingTo" class="replying-to">
        <span>回复 @{{ replyingTo.authorUsername }}</span>
        <q-btn flat dense size="sm" icon="close" @click="replyingTo = null" />
      </div>
      <q-input
        v-model="newCommentContent"
        type="textarea"
        outlined dense
        :placeholder="replyingTo ? `回复 @${replyingTo.authorUsername}` : '写下你的评论...'"
        autogrow
        :maxlength="2000"
        counter
        class="comment-textarea"
      />
      <div class="comment-form-bottom">
        <ImageUploader
          v-model="commentImage"
          entity-type="comment"
          :entity-id="0"
          :max-files="1"
          v-model:uploading="isUploading"
          @upload-success="onUploadSuccess"
        />
        <q-btn
          unelevated color="primary" label="发表评论"
          :loading="submitting"
          :disable="(!newCommentContent.trim() || isUploading)"
          @click="handleSubmit"
        />
      </div>
    </div>

    <!-- Comment list -->
    <div v-if="loading" class="loading-area">
      <q-spinner-dots color="amber" size="32px" />
    </div>
    <div v-else-if="topLevelComments.length" class="comments-list">
      <CommentItem
        v-for="comment in topLevelComments"
        :key="comment.id"
        :comment="comment"
        :all-comments="bountyStore.commentsFlat"
        @reply="setReplyingTo"
        @delete="handleDelete"
      />
    </div>
    <div v-else class="no-comments">
      <p>暂无评论，来说点什么吧</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { useBountyStore } from 'src/stores/bounty'
import CommentItem from './CommentItem.vue'
import ImageUploader from 'src/components/common/ImageUploader.vue'

const props = defineProps({ bountyId: { type: Number, required: true } })

const $q = useQuasar()
const bountyStore = useBountyStore()
const newCommentContent = ref('')
const replyingTo = ref(null)
const submitting = ref(false)
const commentImage = ref('')
const uploadedCommentImage = ref(null)
const isUploading = ref(false)

const loading = computed(() => bountyStore.commentsLoading)
const topLevelComments = computed(() =>
  (bountyStore.comments || []).filter(c => c.parentId == null || c.parentId === '0' || c.parentId === 0)
)
const totalCount = computed(() => (bountyStore.comments || []).length)

onMounted(() => {
  console.log('[BountyComments] onMounted, bountyId:', props.bountyId)
  bountyStore.fetchComments(props.bountyId)
})

function computeReplyPayload(target) {
  if (!target) return { parent_id: 0, reply_to_id: 0 }
  const parent_id = target.parentId ?? target.id
  const reply_to_id = target.id
  return { parent_id, reply_to_id }
}

function setReplyingTo(comment) {
  console.log('[BountyComments] setReplyingTo called, comment:', JSON.stringify(comment))
  replyingTo.value = comment
  console.log('[BountyComments] replyingTo.value is now:', JSON.stringify(replyingTo.value))
  // Scroll to form
  document.querySelector('.comment-form')?.scrollIntoView({ behavior: 'smooth' })
}

function onUploadSuccess(img) {
  console.log('[BountyComments] upload-success received, img:', JSON.stringify(img))
  uploadedCommentImage.value = img
  console.log('[BountyComments] uploadedCommentImage.value is now:', JSON.stringify(uploadedCommentImage.value))
}

async function handleSubmit() {
  submitting.value = true
  try {
    const replyPayload = computeReplyPayload(replyingTo.value)
    const payload = {
      ...replyPayload,
      content: newCommentContent.value,
      imageId: uploadedCommentImage.value?.id ?? 0,
    }
    console.log('[BountyComments] handleSubmit payload:', JSON.stringify(payload))
    console.log('[BountyComments] replyingTo.value:', JSON.stringify(replyingTo.value))
    console.log('[BountyComments] uploadedCommentImage.value:', JSON.stringify(uploadedCommentImage.value))
    await bountyStore.addComment(props.bountyId, payload)
    newCommentContent.value = ''
    replyingTo.value = null
    commentImage.value = ''
    uploadedCommentImage.value = null
  } catch (e) {
    console.error('comment error:', e)
  } finally {
    submitting.value = false
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
      await bountyStore.removeComment(commentId)
      $q.notify({ type: 'positive', message: '评论已删除' })
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '删除失败' })
    }
  })
}
</script>

<style scoped lang="scss">
.comments-section {
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid var(--color-border);
}

.comments-title {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 16px;
}

.comment-form {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 20px;
}

.replying-to {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 6px 12px;
  background: var(--color-bg-elevated);
  border-radius: 6px;
  font-size: 0.85rem;
  color: var(--color-accent-gold);
}

.comment-textarea {
  :deep(.q-field__control) {
    background: var(--color-bg-primary);
    border-color: var(--color-border);
  }
}

.comment-form-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
}

.comment-form-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  gap: 12px;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.no-comments {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
  font-size: 0.9rem;

  p { margin: 0; }
}

.loading-area {
  display: flex;
  justify-content: center;
  padding: 20px;
}
</style>
