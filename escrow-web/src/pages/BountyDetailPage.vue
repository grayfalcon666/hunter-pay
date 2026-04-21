<template>
  <q-page class="detail-page">
    <div class="page-inner">
      <div v-if="bountyStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <div v-else-if="bountyStore.error" class="loading-area" style="flex-direction:column;gap:8px;">
        <p style="color:var(--color-accent-red);">加载失败: {{ bountyStore.error }}</p>
        <q-btn unelevated color="primary" label="重试" @click="bountyStore.fetchBounty(id)" />
      </div>

      <div v-else-if="!bounty" class="loading-area" style="flex-direction:column;gap:8px;">
        <p style="color:var(--color-text-muted);">未找到该悬赏</p>
        <q-btn unelevated color="primary" label="返回" @click="$router.push('/')" />
      </div>

      <template v-else-if="bounty">
        <div class="detail-grid">
          <!-- 左栏：悬赏信息 -->
          <div class="detail-main">

            <!-- 状态条 -->
            <div :class="['detail-status-bar', statusBarClass]" />

            <q-card class="detail-card">
              <q-card-section>
                <div class="detail-header">
                  <StatusBadge :status="bounty.status" />
                  <AmountDisplay :cents="bounty.rewardAmount ?? bounty.reward_amount" class="detail-amount" />
                </div>

                <h1 class="detail-title">{{ bounty.title }}</h1>
                <p class="detail-desc">{{ bounty.description }}</p>

                <q-separator color="border" class="q-my-md" />

                <div class="detail-meta">
                  <div class="meta-item">
                    <span class="meta-label">雇主</span>
                    <router-link :to="`/profile/${bounty.employerUsername ?? bounty.employer_username}`" class="meta-value meta-link">
                      {{ bounty.employerUsername ?? bounty.employer_username }}
                    </router-link>
                  </div>
                  <div class="meta-item" v-if="bountyDeadline">
                    <span class="meta-label">截止时间</span>
                    <span class="meta-value">{{ formatDate(bountyDeadline) }}</span>
                  </div>
                  <div class="meta-item">
                    <span class="meta-label">发布时间</span>
                    <span class="meta-value">{{ formatDate(bounty.created_at) }}</span>
                  </div>
                  <div class="meta-item">
                    <span class="meta-label">悬赏ID</span>
                    <span class="meta-value meta-mono">#{{ bounty.id }}</span>
                  </div>
                  <div class="meta-item" v-if="acceptedHunter">
                    <span class="meta-label">中标猎人</span>
                    <router-link :to="`/profile/${acceptedHunter}`" class="meta-value meta-link">
                      {{ acceptedHunter }}
                    </router-link>
                  </div>
                </div>

                <!-- 提交内容展示 -->
                <div v-if="bounty.submission_text" class="submission-display">
                  <div class="submission-label">工作成果：</div>
                  <p class="submission-content">{{ bounty.submission_text }}</p>
                </div>
              </q-card-section>
            </q-card>

            <!-- 雇主操作面板 -->
            <q-card v-if="isEmployer && bounty.status === 'PENDING'" class="action-card">
              <q-card-section>
                <h3 class="action-title">雇主操作</h3>
                <p class="action-hint">选择一个猎人开始合作</p>
                <div class="action-btns">
                  <q-btn
                    unelevated
                    color="primary"
                    label="确认猎人"
                    :loading="confirmingHunter"
                    @click="showConfirmDialog = true"
                    :disable="!bounty.applications?.length"
                  />
                  <q-btn
                    flat
                    color="negative"
                    label="取消悬赏"
                    :loading="canceling"
                    @click="handleCancel"
                  />
                </div>
              </q-card-section>
            </q-card>

            <!-- 进行中：完成按钮 (旧版保留) -->
            <q-card v-if="isEmployer && bounty.status === 'IN_PROGRESS'" class="action-card action-card--amber">
              <q-card-section>
                <h3 class="action-title">确认完成</h3>
                <p class="action-hint">猎人尚未提交工作成果</p>
              </q-card-section>
            </q-card>

            <!-- 已提交：雇主审核面板 -->
            <q-card v-if="isEmployer && bounty.status === 'SUBMITTED'" class="action-card action-card--amber">
              <q-card-section>
                <h3 class="action-title">审核猎人提交</h3>
                <p class="action-hint">猎人已提交工作成果，请审核</p>
                <div v-if="bounty.submission_text" class="submission-text">
                  <p class="submission-label">提交内容：</p>
                  <p class="submission-content">{{ bounty.submission_text }}</p>
                </div>
                <div class="action-btns q-mt-md">
                  <q-btn
                    unelevated
                    color="positive"
                    icon="check_circle"
                    label="通过"
                    :loading="approving"
                    @click="handleApprove"
                  />
                  <q-btn
                    flat
                    color="negative"
                    icon="cancel"
                    label="拒绝"
                    :loading="rejecting"
                    @click="handleReject"
                  />
                </div>
              </q-card-section>
            </q-card>

            <!-- 被拒绝：猎人可重新提交 -->
            <q-card v-if="isHunter && bounty.status === 'IN_PROGRESS' && hasAppliedHunter" class="action-card action-card--red">
              <q-card-section>
                <h3 class="action-title">提交被拒绝</h3>
                <p class="action-hint">您的提交未通过审核，请修改后重新提交</p>
                <div class="action-btns q-mt-md">
                  <q-btn
                    unelevated
                    color="primary"
                    icon="upload"
                    label="重新提交"
                    :loading="submitting"
                    @click="showSubmitDialog = true"
                  />
                </div>
              </q-card-section>
            </q-card>

            <!-- 失败：删除按钮 -->
            <q-card v-if="isEmployer && bounty.status === 'FAILED'" class="action-card action-card--red">
              <q-card-section>
                <h3 class="action-title">悬赏失败</h3>
                <p class="action-hint">此悬赏因支付失败已终结，可以删除</p>
                <q-btn
                  unelevated
                  color="negative"
                  icon="delete"
                  label="删除悬赏"
                  :loading="deleting"
                  @click="handleDelete"
                />
              </q-card-section>
            </q-card>

            <!-- 猎人操作面板 -->
            <q-card v-if="isHunter && bounty.status === 'PENDING' && !hasApplied" class="action-card">
              <q-card-section>
                <h3 class="action-title">接单</h3>
                <p class="action-hint">接受此悬赏任务</p>
                <q-btn
                  unelevated
                  color="teal"
                  icon="bolt"
                  label="接单"
                  :loading="accepting"
                  @click="handleAccept"
                />
              </q-card-section>
            </q-card>
            <q-card v-else-if="isHunter && bounty.status === 'PENDING' && hasApplied" class="action-card">
              <q-card-section>
                <h3 class="action-title">已申请</h3>
                <p class="action-hint">您已申请此悬赏，请等待雇主确认</p>
              </q-card-section>
            </q-card>

            <!-- 猎人在进行中：提交工作成果 -->
            <q-card v-if="isHunter && bounty.status === 'IN_PROGRESS' && isAcceptedHunter" class="action-card action-card--teal">
              <q-card-section>
                <h3 class="action-title">提交工作成果</h3>
                <p class="action-hint">完成任务后提交您的成果</p>
                <q-btn
                  unelevated
                  color="teal"
                  icon="upload"
                  label="提交成果"
                  :loading="submitting"
                  @click="showSubmitDialog = true"
                />
              </q-card-section>
            </q-card>

            <!-- 猎人已提交：等待审核 -->
            <q-card v-if="isHunter && bounty.status === 'SUBMITTED' && isAcceptedHunter" class="action-card action-card--amber">
              <q-card-section>
                <h3 class="action-title">等待审核</h3>
                <p class="action-hint">您已提交工作成果，请等待雇主审核</p>
                <div v-if="bounty.submission_text" class="submission-text q-mt-sm">
                  <p class="submission-label">您的提交内容：</p>
                  <p class="submission-content">{{ bounty.submission_text }}</p>
                </div>
              </q-card-section>
            </q-card>

            <!-- 评论区 -->
            <BountyComments :bounty-id="Number(id)" />

            <!-- 完成后：互评入口 -->
            <q-card v-if="bounty.status === 'COMPLETED'" class="action-card action-card--teal">
              <q-card-section>
                <h3 class="action-title">合作评价</h3>
                <p v-if="myReviewOfCounterpart" class="action-hint">
                  您已评价对方 ★ {{ myReviewOfCounterpart.rating || 5 }}
                  <span v-if="myReviewOfCounterpart.content || myReviewOfCounterpart.comment">：「{{ myReviewOfCounterpart.content || myReviewOfCounterpart.comment }}」</span>
                </p>
                <p v-else class="action-hint">感谢合作，请为对方留下评价</p>
                <q-btn
                  v-if="!myReviewOfCounterpart"
                  unelevated
                  color="teal"
                  icon="star"
                  label="评价合作方"
                  @click="openReviewDialog"
                />
                <q-btn
                  v-else
                  unelevated
                  color="grey-7"
                  icon="check"
                  label="已评价"
                  disable
                />
              </q-card-section>
            </q-card>

          </div>

          <!-- 右栏：申请列表 -->
          <div class="detail-sidebar">
            <q-card class="applications-card">
              <q-card-section>
                <h3 class="sidebar-title">申请列表</h3>
                <span class="app-count">{{ bounty.applications?.length || 0 }} 人申请</span>
              </q-card-section>

              <q-card-section v-if="!bounty.applications?.length" class="no-apps">
                <q-icon name="inbox" size="32px" color="grey-6" />
                <p>暂无申请</p>
              </q-card-section>

              <q-list v-else separator>
                <q-item v-for="app in bounty.applications" :key="app.id" class="app-item">
                  <q-item-section avatar>
                    <img v-if="app.hunterAvatarUrl" :src="imageUrl(app.hunterAvatarUrl)" class="app-avatar" alt="avatar" />
                    <div v-else class="app-avatar">{{ (app.hunterUsername ?? app.hunter_username)?.[0]?.toUpperCase() ?? '?' }}</div>
                  </q-item-section>
                  <q-item-section>
                    <router-link :to="`/profile/${app.hunterUsername ?? app.hunter_username}`" class="app-name">
                      {{ app.hunterUsername ?? app.hunter_username }}
                    </router-link>
                  </q-item-section>
                  <q-item-section side>
                    <StatusBadge :status="app.status" />
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card>
          </div>
        </div>
      </template>

      <!-- 确认猎人对话框 -->
      <q-dialog v-model="showConfirmDialog">
        <q-card class="confirm-dialog">
          <q-card-section>
            <h3 class="dialog-title">选择猎人</h3>
          </q-card-section>
          <q-card-section>
            <q-list separator>
              <q-item
                v-for="app in bounty?.applications?.filter(a => a.status === 'APPLIED')"
                :key="app.id"
                clickable
                v-ripple
                @click="handleConfirmHunter(app.id)"
              >
                <q-item-section avatar>
                  <img v-if="app.hunterAvatarUrl" :src="imageUrl(app.hunterAvatarUrl)" class="app-avatar" alt="avatar" />
                  <div v-else class="app-avatar">{{ (app.hunterUsername ?? app.hunter_username)?.[0]?.toUpperCase() ?? '?' }}</div>
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ app.hunterUsername ?? app.hunter_username }}</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat label="取消" v-close-popup />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 提交工作成果对话框 -->
      <q-dialog v-model="showSubmitDialog" persistent>
        <q-card class="confirm-dialog">
          <q-card-section>
            <h3 class="dialog-title">提交工作成果</h3>
          </q-card-section>
          <q-card-section>
            <q-input
              v-model="submissionText"
              type="textarea"
              outlined
              autogrow
              placeholder="请描述您完成的工作内容..."
              :maxlength="5000"
              counter
            />
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat label="取消" v-close-popup />
            <q-btn unelevated color="teal" label="提交" :loading="submitting" @click="handleSubmit" />
          </q-card-actions>
        </q-card>
      </q-dialog>

      <!-- 互评对话框 -->
      <q-dialog v-model="showReviewDialog" persistent>
        <q-card class="confirm-dialog">
          <q-card-section>
            <h3 class="dialog-title">评价合作方</h3>
            <p class="action-hint">请对 {{ reviewTarget }} 进行评分</p>
          </q-card-section>
          <q-card-section>
            <div class="review-stars-row">
              <q-btn
                v-for="star in [1, 2, 3, 4, 5]"
                :key="star"
                :icon="reviewRating >= star ? 'star' : 'star_border'"
                :color="reviewRating >= star ? 'amber' : 'grey-7'"
                text-color="grey-9"
                unelevated
                dense
                @click="reviewRating = star"
              />
            </div>
            <q-input
              v-model="reviewComment"
              type="textarea"
              outlined
              autogrow
              placeholder="写点评价吧（选填）..."
              :maxlength="500"
              class="q-mt-md"
            />
          </q-card-section>
          <q-card-actions align="right">
            <q-btn flat label="跳过" v-close-popup />
            <q-btn unelevated color="teal" label="提交评价" :loading="reviewSubmitting" @click="handleSubmitReview" />
          </q-card-actions>
        </q-card>
      </q-dialog>

    </div>
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useBountyStore } from 'src/stores/bounty'
import { useAuthStore } from 'src/stores/auth'
import { useProfileStore } from 'src/stores/profile'
import StatusBadge from 'src/components/bounty/StatusBadge.vue'
import AmountDisplay from 'src/components/bounty/AmountDisplay.vue'
import { imageUrl } from 'src/api/upload'
import BountyComments from 'src/components/bounty/BountyComments.vue'

const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const bountyStore = useBountyStore()
const authStore = useAuthStore()
const profileStore = useProfileStore()

const bounty = computed(() => bountyStore.currentBounty)
const id = computed(() => route.params.id)

const isEmployer = computed(() => (bounty.value?.employerUsername ?? bounty.value?.employer_username) === authStore.username)
const isHunter = computed(() =>
  authStore.isLoggedIn() && !isEmployer.value
)
const hasApplied = computed(() =>
  (bounty.value?.applications || []).some(a => a.hunterUsername === authStore.username || a.hunter_username === authStore.username)
)
const isAcceptedHunter = computed(() =>
  (bounty.value?.applications || []).some(a =>
    (a.hunterUsername === authStore.username || a.hunter_username === authStore.username) &&
    a.status === 'ACCEPTED'
  )
)
const hasAppliedHunter = computed(() =>
  (bounty.value?.applications || []).some(a =>
    (a.hunterUsername === authStore.username || a.hunter_username === authStore.username) &&
    a.status === 'APPLIED'
  )
)
const acceptedHunter = computed(() => {
  const app = (bounty.value?.applications || []).find(a => a.status === 'ACCEPTED')
  return app?.hunterUsername ?? app?.hunter_username ?? null
})

// 兼容 deadline / deadlineTimestamp 两种字段名
const bountyDeadline = computed(() =>
  bounty.value?.deadline ?? bounty.value?.deadlineTimestamp ?? null
)

const showConfirmDialog = ref(false)
const showSubmitDialog = ref(false)
const showReviewDialog = ref(false)
const reviewTarget = ref('')
const reviewRating = ref(5)
const reviewComment = ref('')
const reviewSubmitting = ref(false)
const submissionText = ref('')
const confirmingHunter = ref(false)
const accepting = ref(false)
const canceling = ref(false)
const deleting = ref(false)
const submitting = ref(false)
const approving = ref(false)
const rejecting = ref(false)

const statusBarClass = computed(() => {
  const map = {
    PAYING: 'bar-paying', PENDING: 'bar-pending', IN_PROGRESS: 'bar-progress',
    COMPLETED: 'bar-completed', FAILED: 'bar-failed', CANCELED: 'bar-canceled',
  }
  return map[bounty.value?.status] ?? 'bar-pending'
})

function formatDate(d) {
  if (!d) return '—'
  let ms
  if (d instanceof Date) {
    ms = d.getTime()
  } else if (typeof d === 'number') {
    ms = d < 1e12 ? d * 1000 : d
  } else {
    ms = Date.parse(d)
    if (ms < 1e12 && ms > 1e8) ms *= 1000
  }
  const date = new Date(ms)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN')
}

async function handleAccept() {
  accepting.value = true
  try {
    await bountyStore.acceptBounty(id.value)
    $q.notify({ type: 'positive', message: '接单成功' })
    await bountyStore.fetchBounty(id.value)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '接单失败' })
  } finally {
    accepting.value = false
  }
}

async function handleConfirmHunter(appId) {
  showConfirmDialog.value = false
  confirmingHunter.value = true
  try {
    await bountyStore.confirmHunter(id.value, appId)
    $q.notify({ type: 'positive', message: '已确认猎人，任务开始' })
    await bountyStore.fetchBounty(id.value)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '操作失败' })
  } finally {
    confirmingHunter.value = false
  }
}

async function handleCancel() {
  $q.dialog({
    title: '确认取消',
    message: '确定要取消此悬赏吗？赏金将退还给您。',
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    canceling.value = true
    try {
      await bountyStore.cancelBounty(id.value)
      $q.notify({ type: 'info', message: '悬赏已取消' })
      await bountyStore.fetchBounty(id.value)
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '取消失败' })
    } finally {
      canceling.value = false
    }
  })
}

async function handleSubmit() {
  if (!submissionText.value.trim()) {
    $q.notify({ type: 'warning', message: '请输入提交内容' })
    return
  }
  submitting.value = true
  try {
    await bountyStore.submitBounty(id.value, submissionText.value)
    $q.notify({ type: 'positive', message: '提交成功，等待雇主审核' })
    showSubmitDialog.value = false
    submissionText.value = ''
    await bountyStore.fetchBounty(id.value)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '提交失败' })
  } finally {
    submitting.value = false
  }
}

async function handleApprove() {
  $q.dialog({
    title: '确认通过',
    message: '确定要通过猎人的提交吗？赏金将支付给猎人。',
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    approving.value = true
    try {
      await bountyStore.approveBounty(id.value)
      $q.notify({ type: 'positive', message: '已通过，赏金已支付' })
      await bountyStore.fetchBounty(id.value)
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '操作失败' })
    } finally {
      approving.value = false
    }
  })
}

async function handleReject() {
  $q.dialog({
    title: '确认拒绝',
    message: '确定要拒绝猎人的提交吗？猎人可以修改后重新提交。',
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    rejecting.value = true
    try {
      await bountyStore.rejectBounty(id.value)
      $q.notify({ type: 'info', message: '已拒绝，猎人可重新提交' })
      await bountyStore.fetchBounty(id.value)
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '操作失败' })
    } finally {
      rejecting.value = false
    }
  })
}

async function handleDelete() {
  $q.dialog({
    title: '确认删除',
    message: '确定要删除此悬赏吗？此操作不可恢复。',
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    deleting.value = true
    try {
      await bountyStore.deleteBounty(id.value)
      $q.notify({ type: 'positive', message: '悬赏已删除' })
      // Redirect to hall
      router.push('/')
    } catch (e) {
      $q.notify({ type: 'negative', message: e.message || '删除失败' })
    } finally {
      deleting.value = false
    }
  })
}

onMounted(() => {
  console.log('[BountyDetail] onMounted, id:', id.value)
  bountyStore.resetState()
  bountyStore.fetchBounty(id.value)
  // 拉取当前用户评论列表，用于判断是否已评价
  if (authStore.isLoggedIn()) {
    profileStore.fetchReviews(authStore.username)
  }
})

watch(id, () => { bountyStore.resetState(); bountyStore.fetchBounty(id.value) })

// 计算当前用户对该 bounty 对端评价
const myReviewOfCounterpart = computed(() => {
  if (!authStore.isLoggedIn()) return null
  const myUsername = authStore.username
  const employer = bounty.value?.employerUsername ?? bounty.value?.employer_username
  const hunter = bounty.value?.hunterUsername ?? bounty.value?.hunter_username
  // 找到我作为 reviewer 的评价记录
  return profileStore.reviews.find(r =>
    (r.reviewer_username || r.reviewerUsername) === myUsername &&
    (r.bounty_id || r.bountyId) === Number(id.value)
  ) || null
})

function openReviewDialog() {
  const myUsername = authStore.username
  const employer = bounty.value?.employerUsername ?? bounty.value?.employer_username
  const hunter = bounty.value?.hunterUsername ?? bounty.value?.hunter_username ?? acceptedHunter.value
  if (myUsername === employer) {
    reviewTarget.value = hunter
  } else {
    reviewTarget.value = employer
  }
  showReviewDialog.value = true
}

async function handleSubmitReview() {
  if (!reviewRating.value) {
    $q.notify({ type: 'warning', message: '请选择评分' })
    return
  }
  reviewSubmitting.value = true
  try {
    await bountyStore.submitReview({
      reviewedUsername: reviewTarget.value,
      bountyId: id.value,
      rating: reviewRating.value,
      comment: reviewComment.value,
    })
    $q.notify({ type: 'positive', message: '评价已提交，感谢您的反馈' })
    showReviewDialog.value = false
    reviewRating.value = 5
    reviewComment.value = ''
    reviewTarget.value = ''
    // 刷新评论列表，同步 hasReviewed 状态
    profileStore.fetchReviews(authStore.username)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '提交评价失败' })
  } finally {
    reviewSubmitting.value = false
  }
}
</script>

<style scoped lang="scss">
.detail-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 1280px; margin: 0 auto; padding: 48px 24px; }
.loading-area { display: flex; justify-content: center; padding: 80px; }

.detail-status-bar { height: 3px; border-radius: 2px; margin-bottom: 24px; }
.bar-paying    { background: var(--color-accent-gold); }
.bar-pending  { background: var(--color-accent-teal); }
.bar-progress { background: var(--color-accent-amber); }
.bar-submitted { background: var(--color-accent-amber); }
.bar-completed { background: var(--color-accent-green); }
.bar-rejected { background: var(--color-accent-red); }
.bar-failed, .bar-canceled, .bar-expired  { background: var(--color-accent-red); }

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
  align-items: start;
  @media (max-width: 900px) { grid-template-columns: 1fr; }
}

.detail-card, .action-card, .applications-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.detail-amount { font-size: 1.4rem; }

.detail-title {
  font-family: var(--font-display);
  font-size: 1.8rem;
  font-weight: 700;
  line-height: 1.2;
  margin-bottom: 12px;
}

.detail-desc {
  color: var(--color-text-muted);
  font-size: 0.95rem;
  line-height: 1.7;
  white-space: pre-wrap;
  margin: 0;
}

.detail-meta {
  display: flex;
  gap: 32px;
  flex-wrap: wrap;
}

.meta-item { display: flex; flex-direction: column; gap: 4px; }
.meta-label { font-size: 0.75rem; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.08em; font-family: var(--font-mono); }
.meta-value { font-size: 0.9rem; font-weight: 500; color: var(--color-text-primary); }
.meta-link { color: var(--color-accent-teal); text-decoration: none; &:hover { text-decoration: underline; } }
.meta-mono { font-family: var(--font-mono); }

.action-card {
  margin-top: 16px;
  &.action-card--amber { border-color: rgba(251,191,36,0.3) !important; }
  &.action-card--red { border-color: rgba(239,68,68,0.3) !important; }
  &.action-card--teal { border-color: rgba(6,214,160,0.3) !important; }
}

.submission-text, .submission-display {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 12px;
  margin-top: 8px;
}
.submission-label { font-size: 0.78rem; color: var(--color-text-muted); margin-bottom: 6px; font-family: var(--font-mono); }
.submission-content { font-size: 0.9rem; color: var(--color-text-primary); white-space: pre-wrap; margin: 0; }
.action-title {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 4px;
}
.action-hint { font-size: 0.85rem; color: var(--color-text-muted); margin-bottom: 16px; }
.action-btns { display: flex; gap: 12px; flex-wrap: wrap; }

.sidebar-title {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 600;
}
.app-count { font-size: 0.8rem; color: var(--color-text-muted); margin-left: 8px; }
.no-apps { text-align: center; padding: 32px; color: var(--color-text-muted); p { margin: 8px 0 0; font-size: 0.85rem; } }

.app-item { border-radius: 8px; margin: 4px 0; }
.app-avatar {
  width: 32px; height: 32px; border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display); font-size: 0.85rem; font-weight: 600;
  color: var(--color-accent-gold);
  object-fit: cover;
}

img.app-avatar {
  display: block;
}
.app-name { color: var(--color-accent-teal); text-decoration: none; font-weight: 500; &:hover { text-decoration: underline; } }

.confirm-dialog {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  min-width: 320px;
}
.dialog-title { font-family: var(--font-display); font-size: 1.2rem; font-weight: 600; }
.review-stars-row { display: flex; gap: 4px; justify-content: center; padding: 8px 0; }
</style>
