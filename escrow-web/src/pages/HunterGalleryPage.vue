<template>
  <q-page class="hunter-gallery-page">
    <div class="page-inner">

      <!-- 页面标题 -->
      <div class="page-hero">
        <h1 class="hero-title">猎人招募</h1>
        <p class="hero-subtitle">发现优秀猎人，发起接单邀请</p>
      </div>

      <!-- 搜索栏 -->
      <div class="search-bar">
        <q-input
          v-model="searchQuery"
          placeholder="搜索猎人用户名..."
          outlined
          dense
          class="search-input"
          @keyup.enter="doSearch"
        >
          <template #prepend>
            <q-icon name="search" />
          </template>
          <template #append>
            <q-btn v-if="searchQuery" flat dense round icon="clear" @click="clearSearch" />
          </template>
        </q-input>
        <q-btn unelevated color="gold" label="搜索" @click="doSearch" class="search-btn" />
      </div>

      <!-- 筛选排序 -->
      <div class="filter-bar">
        <div class="filter-tabs">
          <button
            v-for="opt in sortOptions"
            :key="opt.value"
            :class="['filter-tab', { active: sortBy === opt.value }]"
            @click="sortBy = opt.value; doSearch()"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-area">
        <q-spinner-dots color="gold" size="40px" />
      </div>

      <!-- 猎人列表 -->
      <div v-else-if="hunters.length" class="hunter-grid">
        <q-card
          v-for="(hunter, i) in hunters"
          :key="hunter.username"
          class="hunter-card card-reveal"
          :style="{ animationDelay: `${i * 60}ms` }"
          @click="openHunterDrawer(hunter)"
        >
          <div class="status-bar bar-hunter" />
          <q-card-section class="card-content">
            <div class="card-header">
              <img v-if="hunter.avatarUrl" :src="imageUrl(hunter.avatarUrl)" class="hunter-avatar" alt="avatar" />
              <div v-else class="hunter-avatar">{{ (hunter.full_name || hunter.username || '?')[0].toUpperCase() }}</div>
              <div class="hunter-info">
                <h3 class="hunter-name">{{ hunter.full_name || hunter.username }}</h3>
                <span class="hunter-username">@{{ hunter.username }}</span>
              </div>
            </div>
            <p v-if="hunter.bio" class="hunter-bio">{{ hunter.bio }}</p>
            <q-separator color="border" class="q-my-md" />
            <div class="hunter-stats">
              <div class="stat">
                <span class="stat-value">{{ hunter.totalBountiesCompleted || 0 }}</span>
                <span class="stat-label">完成</span>
              </div>
              <div class="stat">
                <span class="stat-value">{{ displayPercent(hunter.goodReviewRate ?? hunter.good_review_rate) }}</span>
                <span class="stat-label">好评率</span>
              </div>
              <div class="stat">
                <span class="stat-value">{{ hunter.hunterFulfillmentIndex ?? '—' }}</span>
                <span class="stat-label">履约</span>
              </div>
            </div>
          </q-card-section>
        </q-card>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">◈</div>
        <h3>暂无猎人</h3>
        <p>系统中的猎人信息将显示在这里</p>
      </div>

    </div>

    <!-- 猎人详情抽屉 -->
    <q-dialog v-model="drawerOpen" position="right" full-height>
      <q-card class="hunter-drawer" v-if="selectedHunter">
        <q-card-section class="drawer-header">
          <img v-if="selectedHunter.avatarUrl" :src="imageUrl(selectedHunter.avatarUrl)" class="drawer-avatar" alt="avatar" />
          <div v-else class="drawer-avatar">{{ (selectedHunter.full_name || selectedHunter.username || '?')[0].toUpperCase() }}</div>
          <div class="drawer-title">
            <h3>{{ selectedHunter.full_name || selectedHunter.username }}</h3>
            <span>@{{ selectedHunter.username }}</span>
          </div>
          <q-btn flat dense round icon="close" class="close-btn" v-close-popup />
        </q-card-section>

        <q-separator color="border" />

        <q-card-section v-if="selectedHunter.bio">
          <div class="drawer-label">简介</div>
          <p class="drawer-bio">{{ selectedHunter.bio }}</p>
        </q-card-section>

        <q-card-section>
          <div class="drawer-label">统计</div>
          <div class="drawer-stats">
            <div class="d-stat">
              <span class="d-val">{{ selectedHunter.totalBountiesCompleted || 0 }}</span>
              <span class="d-key">完成</span>
            </div>
            <div class="d-stat">
              <span class="d-val">{{ displayPercent(selectedHunter.goodReviewRate ?? selectedHunter.good_review_rate) }}</span>
              <span class="d-key">好评率</span>
            </div>
            <div class="d-stat">
              <span class="d-val">{{ formatYuan(selectedHunter.totalEarnings) }}</span>
              <span class="d-key">总收入</span>
            </div>
            <div class="d-stat">
              <span class="d-val">{{ selectedHunter.hunterFulfillmentIndex ?? '—' }}</span>
              <span class="d-key">履约</span>
            </div>
          </div>
        </q-card-section>

        <q-card-section v-if="selectedHunter.workLocation || selectedHunter.experienceLevel">
          <div class="drawer-label">信息</div>
          <div class="drawer-meta">
            <span v-if="selectedHunter.workLocation" class="meta-tag">{{ selectedHunter.workLocation }}</span>
            <span v-if="selectedHunter.experienceLevel" class="meta-tag">{{ selectedHunter.experienceLevel }}</span>
          </div>
        </q-card-section>

        <q-separator color="border" />

        <q-card-actions class="drawer-actions">
          <q-btn
            unelevated
            color="gold"
            icon="mail"
            label="发消息"
            class="action-btn"
            @click="openChat"
          />
          <q-btn
            unelevated
            color="primary"
            icon="person"
            label="查看资料"
            class="action-btn"
            @click="router.push(`/profile/${selectedHunter.username}`)"
          />
          <q-btn
            unelevated
            color="teal"
            icon="send"
            label="邀请接单"
            class="action-btn"
            :disable="!selectedBountyId"
            @click="showInviteDialog = true"
          />
        </q-card-actions>

        <!-- 悬赏选择（用于邀请） -->
        <q-expansion-item label="选择悬赏（邀请接单用）" class="bounty-selector">
          <div v-if="myBounties.length === 0" class="no-bounties">
            暂无可邀请的悬赏（仅 PENDING 状态的悬赏可邀请）
          </div>
          <div
            v-for="b in myBounties"
            :key="b.id"
            :class="['bounty-option', { selected: selectedBountyId === b.id }]"
            @click="selectedBountyId = b.id"
          >
            <span class="bounty-title">{{ b.title }}</span>
            <span class="bounty-amount">{{ formatYuan(b.rewardAmount ?? b.reward_amount) }}</span>
          </div>
        </q-expansion-item>
      </q-card>
    </q-dialog>

    <!-- 邀请确认弹窗 -->
    <q-dialog v-model="showInviteDialog">
      <q-card class="confirm-dialog">
        <q-card-section>
          <h3 class="dialog-title">确认邀请</h3>
          <p class="dialog-desc">
            邀请 <strong>{{ selectedHunter?.username }}</strong> 接单「{{ selectedBountyTitle }}」？
          </p>
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat label="取消" v-close-popup />
          <q-btn unelevated color="teal" label="确认邀请" @click="confirmInvite" :loading="inviteLoading" />
        </q-card-actions>
      </q-card>
    </q-dialog>

  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useAuthStore } from 'src/stores/auth'
import { useChatStore } from 'src/stores/chat'
import { useBountyStore } from 'src/stores/bounty'
import apiClient from 'src/api/client'
import { createInvitation } from 'src/api/invitation'
import { imageUrl } from 'src/api/upload'

const $q = useQuasar()
const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()
const bountyStore = useBountyStore()

const loading = ref(false)
const hunters = ref([])
const searchQuery = ref('')
const sortBy = ref('good_review_rate')
const drawerOpen = ref(false)
const selectedHunter = ref(null)
const showInviteDialog = ref(false)
const inviteLoading = ref(false)
const myBounties = ref([])
const selectedBountyId = ref(null)

const sortOptions = [
  { label: '好评率', value: 'good_review_rate' },
  { label: '履约', value: 'fulfillment' },
  { label: '完成量', value: 'total_bounties_completed' },
]

const selectedBountyTitle = computed(() => {
  const b = myBounties.value.find(b => b.id === selectedBountyId.value)
  return b ? b.title : ''
})

async function doSearch() {
  loading.value = true
  try {
    const params = { query: searchQuery.value || '', limit: 30, sort_by: sortBy.value }
    const data = await apiClient.get('/hunters/search', { params })
    let list = data.hunters || []
    if (sortBy.value === 'good_review_rate') {
      list = [...list].sort((a, b) => (b.goodReviewRate ?? b.good_review_rate ?? 0) - (a.goodReviewRate ?? a.good_review_rate ?? 0))
    } else if (sortBy.value === 'fulfillment') {
      list = [...list].sort((a, b) => (b.hunterFulfillmentIndex ?? 50) - (a.hunterFulfillmentIndex ?? 50))
    } else if (sortBy.value === 'total_bounties_completed') {
      list = [...list].sort((a, b) => (b.totalBountiesCompleted ?? b.total_bounties_completed) - (a.totalBountiesCompleted ?? a.total_bounties_completed))
    }
    hunters.value = list
  } catch (e) {
    console.error('search error:', e)
  } finally {
    loading.value = false
  }
}

function clearSearch() {
  searchQuery.value = ''
  doSearch()
}

function openHunterDrawer(hunter) {
  selectedHunter.value = hunter
  selectedBountyId.value = null
  loadMyBounties()
  drawerOpen.value = true
}

async function loadMyBounties() {
  try {
    await bountyStore.fetchBounties({ status: 'PENDING', pageSize: 50 })
    // 只显示当前用户发布的悬赏
    myBounties.value = (bountyStore.bounties || []).filter(b =>
      b.employerUsername === authStore.username ||
      b.employer_username === authStore.username
    )
  } catch (e) {
    console.error('loadMyBounties error:', e)
  }
}

function openChat() {
  if (!selectedHunter.value) return
  chatStore.startConversation(selectedHunter.value.username)
}

async function confirmInvite() {
  if (!selectedBountyId.value || !selectedHunter.value) return
  inviteLoading.value = true
  try {
    await createInvitation(selectedBountyId.value, selectedHunter.value.username)
    $q.notify({ type: 'positive', message: '邀请已发送，等待猎人响应' })
    showInviteDialog.value = false
    drawerOpen.value = false
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '邀请失败' })
  } finally {
    inviteLoading.value = false
  }
}

function displayPercent(val) {
  if (!val && val !== 0) return '—'
  return `${(Number(val) * 100).toFixed(0)}%`
}

function formatYuan(cents) {
  const n = typeof cents === 'string' ? parseInt(cents, 10) : cents
  if (isNaN(n)) return '¥ 0.00'
  return `¥ ${(n / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`
}

onMounted(() => {
  doSearch()
})
</script>

<style scoped lang="scss">
.hunter-gallery-page {
  min-height: 100vh;
  background: var(--color-bg-primary);
}

.page-inner {
  max-width: 1280px;
  margin: 0 auto;
  padding: 48px 24px;
}

.page-hero {
  text-align: center;
  margin-bottom: 48px;
}

.hero-title {
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3.5rem);
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: 0.05em;
  text-transform: uppercase;
  margin-bottom: 12px;
}

.hero-subtitle {
  font-size: 1rem;
  color: var(--color-text-muted);
  margin: 0;
}

.search-bar {
  display: flex;
  gap: 12px;
  max-width: 600px;
  margin: 0 auto 32px;
}

.search-input {
  flex: 1;
}

.search-btn {
  font-weight: 600;
  background: var(--color-accent-gold) !important;
  color: #0b0d12 !important;
}

.filter-bar {
  margin-bottom: 32px;
}

.filter-tabs {
  display: flex;
  gap: 4px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 4px;
  width: fit-content;
}

.filter-tab {
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: 0.875rem;
  font-weight: 500;
  padding: 8px 16px;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover { color: var(--color-text-primary); background: var(--color-bg-elevated); }
  &.active {
    background: var(--color-accent-gold);
    color: #0b0d12;
    font-weight: 600;
  }
}

.loading-area {
  display: flex;
  justify-content: center;
  padding: 80px;
}

.empty-state {
  text-align: center;
  padding: 80px 24px;
  .empty-icon { font-size: 4rem; color: var(--color-border); margin-bottom: 16px; }
  h3 { font-family: var(--font-display); font-size: 1.5rem; color: var(--color-text-muted); margin-bottom: 8px; }
  p { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }
}

.hunter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.hunter-card {
  position: relative;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  flex-direction: column;

  &:hover {
    border-color: var(--color-accent-teal) !important;
    box-shadow: 0 8px 40px rgba(0,0,0,0.5), 0 0 0 1px var(--color-glow-teal);
  }
}

.status-bar {
  height: 3px;
  flex-shrink: 0;
}
.bar-hunter {
  background: var(--color-accent-teal);
}

.card-content {
  display: flex;
  flex-direction: column;
  flex: 1;
  padding: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.hunter-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-accent-teal);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--color-accent-teal);
  box-shadow: 0 0 12px var(--color-glow-teal);
  flex-shrink: 0;
  object-fit: cover;
}

img.hunter-avatar {
  display: block;
}

.hunter-info {
  flex: 1;
  min-width: 0;
}

.hunter-name {
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hunter-username {
  font-size: 0.78rem;
  color: var(--color-text-muted);
  font-family: var(--font-mono);
}

.hunter-bio {
  font-size: 0.85rem;
  color: var(--color-text-muted);
  line-height: 1.4;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  margin: 0;
}

.hunter-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-value {
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--color-accent-teal);
}

.stat-label {
  font-size: 0.6rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-family: var(--font-mono);
}

// 抽屉样式
.hunter-drawer {
  width: 380px;
  max-width: 100vw;
  background: var(--color-bg-secondary) !important;
  border-left: 1px solid var(--color-border);
  border-radius: 0 !important;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.drawer-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}

.drawer-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-accent-teal);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 1.4rem;
  font-weight: 700;
  color: var(--color-accent-teal);
  flex-shrink: 0;
  object-fit: cover;
}

img.drawer-avatar {
  display: block;
}

.drawer-title {
  flex: 1;
  h3 { font-family: var(--font-display); font-size: 1.1rem; font-weight: 700; margin-bottom: 2px; }
  span { font-size: 0.8rem; color: var(--color-text-muted); font-family: var(--font-mono); }
}

.close-btn { color: var(--color-text-muted); }

.drawer-label {
  font-size: 0.7rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-family: var(--font-mono);
  margin-bottom: 8px;
}

.drawer-bio {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  line-height: 1.5;
  margin: 0;
}

.drawer-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.d-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.d-val {
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--color-accent-teal);
}

.d-key {
  font-size: 0.6rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  font-family: var(--font-mono);
}

.drawer-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-tag {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  padding: 3px 8px;
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.drawer-actions {
  padding: 16px;
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  font-weight: 600;
}

.bounty-selector {
  background: var(--color-bg-elevated);
}

.no-bounties {
  padding: 16px;
  font-size: 0.85rem;
  color: var(--color-text-muted);
  text-align: center;
}

.bounty-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.15s;
  border-bottom: 1px solid var(--color-border);

  &:hover { background: var(--color-bg-secondary); }
  &.selected { background: rgba(45, 212, 191, 0.1); border-left: 3px solid var(--color-accent-teal); }
}

.bounty-title { font-size: 0.85rem; font-weight: 500; }
.bounty-amount { font-family: var(--font-mono); font-weight: 700; color: var(--color-accent-gold); font-size: 0.85rem; }

// 确认弹窗
.confirm-dialog {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
  min-width: 320px;
}

.dialog-title { font-family: var(--font-display); font-size: 1.1rem; font-weight: 700; margin-bottom: 12px; }
.dialog-desc { font-size: 0.9rem; color: var(--color-text-muted); margin: 0; line-height: 1.5; }
</style>
