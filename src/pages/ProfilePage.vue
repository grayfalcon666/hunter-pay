<template>
  <q-page class="profile-page">
    <div class="page-inner">

      <div v-if="profileStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <template v-else-if="profileStore.profile">
        <div class="profile-grid">

          <!-- 左侧：头像 + 基本信息 -->
          <div class="profile-aside">
            <q-card class="profile-card">
              <q-card-section class="profile-avatar-section">
                <div class="avatar-wrapper">
                  <div class="avatar-circle">
                    {{ (profileStore.profile.full_name || profileStore.profile.username)[0].toUpperCase() }}
                  </div>
                </div>
                <h2 class="profile-name">{{ profileStore.profile.full_name || profileStore.profile.username }}</h2>
                <span class="profile-username">@{{ profileStore.profile.username }}</span>
                <p v-if="profileStore.profile.bio" class="profile-bio">{{ profileStore.profile.bio }}</p>
              </q-card-section>

              <q-separator color="border" />

              <q-card-section>
                <div class="stat-grid">
                  <div class="stat-item">
                    <span class="stat-value text-amount">{{ formatYuan(profileStore.profile.totalEarnings) }}</span>
                    <span class="stat-label">总收入</span>
                  </div>
                  <div class="stat-item">
                    <span class="stat-value">{{ profileStore.profile.totalBountiesPosted }}</span>
                    <span class="stat-label">发布</span>
                  </div>
                  <div class="stat-item">
                    <span class="stat-value">{{ profileStore.profile.totalBountiesCompleted ?? profileStore.profile.total_bounties_completed }}</span>
                    <span class="stat-label">完成(猎)</span>
                  </div>
                  <div class="stat-item">
                    <span class="stat-value">{{ profileStore.profile.totalBountiesCompletedAsEmployer ?? profileStore.profile.total_bounties_completed_as_employer }}</span>
                    <span class="stat-label">完成(雇)</span>
                  </div>
                </div>
                <div class="fulfillment-row">
                  <div class="fulfillment-item">
                    <q-linear-progress :value="profileStore.profile.hunterFulfillmentIndex / 100" color="teal" track-color="grey-8" size="8px" style="width: 80px;" />
                    <span class="fulfillment-value text-teal">{{ profileStore.profile.hunterFulfillmentIndex ?? 50 }}</span>
                    <span class="stat-label">猎人履约</span>
                  </div>
                  <div class="fulfillment-item">
                    <q-linear-progress :value="profileStore.profile.employerFulfillmentIndex / 100" color="amber" track-color="grey-8" size="8px" style="width: 80px;" />
                    <span class="fulfillment-value text-amber">{{ profileStore.profile.employerFulfillmentIndex ?? 50 }}</span>
                    <span class="stat-label">雇主履约</span>
                  </div>
                </div>
                <div class="rating-row">
                  <span class="stat-value" :class="ratingClass">
                    {{ displayRating }}
                  </span>
                  <span class="stat-label">好评率</span>
                </div>
              </q-card-section>

              <q-card-actions align="center">
                <q-btn v-if="isOwn" flat label="编辑资料" icon="edit" to="/profile/edit" />
                <q-btn
                  v-if="!isOwn && authStore.isLoggedIn()"
                  unelevated color="primary" label="发消息" icon="chat"
                  @click="openChat"
                />
              </q-card-actions>
            </q-card>
          </div>

          <!-- 右侧：评价列表 -->
          <div class="profile-main">
            <q-card class="reviews-card">
              <q-card-section class="reviews-header">
                <h3 class="reviews-title">用户评价</h3>
                <router-link :to="`/reviews/${profileStore.profile.username}`" class="see-all">
                  查看全部 →
                </router-link>
              </q-card-section>

              <div v-if="profileStore.loading" class="loading-area">
                <q-spinner-dots color="amber" size="32px" />
              </div>
              <div v-else-if="profileStore.reviews.length" class="reviews-list">
                <div v-for="review in profileStore.reviews.slice(0, 5)" :key="review.id" class="review-item">
                  <div class="review-header">
                    <div class="reviewer-avatar">{{ (review.reviewerUsername || '?')[0].toUpperCase() }}</div>
                    <div>
                      <router-link :to="`/profile/${review.reviewerUsername}`" class="reviewer-name">
                        {{ review.reviewerUsername }}
                      </router-link>
                      <div class="review-stars">★ {{ review.rating || 5 }}</div>
                    </div>
                    <span class="review-date">{{ formatDate(review.createdAt) }}</span>
                  </div>
                  <p v-if="review.comment" class="review-content">{{ review.comment }}</p>
                </div>
              </div>
              <q-card-section v-else class="no-reviews">
                <p>暂无评价</p>
              </q-card-section>
            </q-card>
          </div>

        </div>
      </template>

    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProfileStore } from 'src/stores/profile'
import { useAuthStore } from 'src/stores/auth'
import { useChatStore } from 'src/stores/chat'

const route = useRoute()
const profileStore = useProfileStore()
const authStore = useAuthStore()
const chatStore = useChatStore()

const username = computed(() => route.params.username)
const isOwn = computed(() => username.value === authStore.username)
const ratingClass = computed(() => {
  const r = profileStore.profile?.goodReviewRate
  if (r >= 0.8) return 'rating-high'
  if (r >= 0.5) return 'rating-mid'
  return 'rating-low'
})

const displayRating = computed(() => {
  const r = profileStore.profile?.goodReviewRate
  if (r === undefined || r === null || r === 0) return '—'
  return (r * 100).toFixed(0) + '%'
})

function formatYuan(cents) {
  const n = typeof cents === 'string' ? parseInt(cents, 10) : cents
  if (isNaN(n)) return '¥ 0.00'
  return `¥ ${(n / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`
}

function formatDate(d) {
  if (!d) return '—'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('zh-CN')
}

async function openChat() {
  await chatStore.startConversation(username.value)
}

onMounted(async () => {
  await profileStore.fetchProfile(username.value)
  await profileStore.fetchReviews(username.value)
})
</script>

<style scoped lang="scss">
.profile-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 1100px; margin: 0 auto; padding: 48px 24px; }
.loading-area { display: flex; justify-content: center; padding: 80px; }

.profile-grid {
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: 24px;
  align-items: start;
  @media (max-width: 768px) { grid-template-columns: 1fr; }
}

.profile-card, .reviews-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.profile-avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.avatar-wrapper { margin-bottom: 16px; }
.avatar-circle {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-accent-gold);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 2.2rem;
  font-weight: 700;
  color: var(--color-accent-gold);
  box-shadow: 0 0 24px var(--color-glow-gold);
}

.profile-name { font-family: var(--font-display); font-size: 1.4rem; font-weight: 700; margin-bottom: 4px; }
.profile-username { font-size: 0.85rem; color: var(--color-text-muted); font-family: var(--font-mono); }
.profile-bio { font-size: 0.9rem; color: var(--color-text-muted); margin-top: 12px; line-height: 1.5; }

.stat-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.rating-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-family: var(--font-display);
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--color-text-primary);
  &.text-amount { color: var(--color-accent-gold); }
}

.rating-high { color: var(--color-accent-green) !important; }
.rating-mid  { color: var(--color-accent-amber) !important; }
.rating-low  { color: var(--color-accent-red) !important; }

.stat-label { font-size: 0.72rem; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.08em; font-family: var(--font-mono); }

.fulfillment-row {
  display: flex;
  justify-content: space-around;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
  margin-top: 12px;
}

.fulfillment-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.fulfillment-value {
  font-family: var(--font-display);
  font-size: 1.1rem;
  font-weight: 700;
}

.reviews-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.reviews-title { font-family: var(--font-display); font-size: 1.1rem; font-weight: 600; margin: 0; }
.see-all { color: var(--color-accent-teal); text-decoration: none; font-size: 0.85rem; &:hover { text-decoration: underline; } }

.reviews-list { padding: 0 16px 16px; }

.review-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--color-border);
  &:last-child { border-bottom: none; }
}

.review-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.reviewer-avatar {
  width: 28px; height: 28px; border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display); font-size: 0.75rem; font-weight: 600;
  color: var(--color-accent-gold);
}

.reviewer-name { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); text-decoration: none; }
.review-stars { font-size: 0.75rem; color: var(--color-accent-amber); }
.review-date { margin-left: auto; font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono); }
.review-content { font-size: 0.875rem; color: var(--color-text-muted); margin: 0; line-height: 1.5; }
.no-reviews { text-align: center; color: var(--color-text-muted); padding: 32px; p { margin: 0; } }
</style>
