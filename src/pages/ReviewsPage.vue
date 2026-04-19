<template>
  <q-page class="reviews-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">{{ username }} 的评价</h1>
      </div>

      <div v-if="profileStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <template v-else>
        <q-tabs v-model="activeTab" class="review-tabs" active-color="amber" indicator-color="amber" align="left" narrow-indicator>
          <q-tab name="hunter" label="身为猎人" />
          <q-tab name="employer" label="身为雇主" />
        </q-tabs>

        <q-tab-panels v-model="activeTab" animated class="review-panels">
          <q-tab-panel name="hunter" class="review-panel">
            <div v-if="hunterReviews.length" class="reviews-list">
              <q-card v-for="review in hunterReviews" :key="review.id" class="review-card">
                <q-card-section>
                  <div class="review-header">
                    <div class="reviewer-avatar">
                      {{ (review.reviewerUsername || '?')[0].toUpperCase() }}
                    </div>
                    <div class="reviewer-info">
                      <router-link :to="`/profile/${review.reviewerUsername}`" class="reviewer-name">
                        {{ review.reviewerUsername }}
                      </router-link>
                      <div class="review-stars">
                        <span v-for="n in 5" :key="n" :class="['star', { filled: n <= (review.rating || 5) }]">★</span>
                      </div>
                    </div>
                    <span class="review-date">{{ formatDate(review.createdAt) }}</span>
                  </div>
                  <p v-if="review.comment" class="review-content">{{ review.comment }}</p>
                  <div v-if="review.bountyId" class="bounty-ref">悬赏 #{{ review.bountyId }}</div>
                </q-card-section>
              </q-card>
            </div>
            <div v-else class="empty-state">
              <p>暂无猎人评价</p>
            </div>
          </q-tab-panel>

          <q-tab-panel name="employer" class="review-panel">
            <div v-if="employerReviews.length" class="reviews-list">
              <q-card v-for="review in employerReviews" :key="review.id" class="review-card">
                <q-card-section>
                  <div class="review-header">
                    <div class="reviewer-avatar">
                      {{ (review.reviewerUsername || '?')[0].toUpperCase() }}
                    </div>
                    <div class="reviewer-info">
                      <router-link :to="`/profile/${review.reviewerUsername}`" class="reviewer-name">
                        {{ review.reviewerUsername }}
                      </router-link>
                      <div class="review-stars">
                        <span v-for="n in 5" :key="n" :class="['star', { filled: n <= (review.rating || 5) }]">★</span>
                      </div>
                    </div>
                    <span class="review-date">{{ formatDate(review.createdAt) }}</span>
                  </div>
                  <p v-if="review.comment" class="review-content">{{ review.comment }}</p>
                  <div v-if="review.bountyId" class="bounty-ref">悬赏 #{{ review.bountyId }}</div>
                </q-card-section>
              </q-card>
            </div>
            <div v-else class="empty-state">
              <p>暂无雇主评价</p>
            </div>
          </q-tab-panel>
        </q-tab-panels>
      </template>
    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useProfileStore } from 'src/stores/profile'

const route = useRoute()
const profileStore = useProfileStore()
const username = computed(() => route.params.username)
const activeTab = ref('hunter')

const hunterReviews = computed(() =>
  profileStore.reviews.filter(r => r.reviewType === 'EMPLOYER_TO_HUNTER')
)
const employerReviews = computed(() =>
  profileStore.reviews.filter(r => r.reviewType === 'HUNTER_TO_EMPLOYER')
)

function formatDate(d) {
  if (!d) return '—'
  const date = new Date(d)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('zh-CN')
}

onMounted(() => profileStore.fetchReviews(username.value))
</script>

<style scoped lang="scss">
.reviews-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 860px; margin: 0 auto; padding: 48px 24px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; }
.loading-area, .empty-state { display: flex; justify-content: center; padding: 80px; color: var(--color-text-muted); }

.review-tabs {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  margin-bottom: 0;
}

.review-panels {
  background: transparent;
}

.review-panel { padding: 16px 0 0 0; }

.reviews-list { display: flex; flex-direction: column; gap: 16px; }

.review-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.review-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.reviewer-avatar {
  width: 36px; height: 36px; border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display); font-size: 0.9rem; font-weight: 600;
  color: var(--color-accent-gold);
  flex-shrink: 0;
}

.reviewer-name { font-size: 0.9rem; font-weight: 600; text-decoration: none; color: var(--color-text-primary); &:hover { color: var(--color-accent-teal); } }

.review-stars {
  display: flex;
  gap: 2px;
  .star { font-size: 0.8rem; color: var(--color-border); &.filled { color: var(--color-accent-amber); } }
}

.review-date { margin-left: auto; font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono); }
.review-content { font-size: 0.9rem; color: var(--color-text-muted); line-height: 1.6; margin: 0; }
.bounty-ref { font-size: 0.75rem; color: var(--color-text-muted); margin-top: 8px; font-family: var(--font-mono); }
</style>
