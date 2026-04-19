<template>
  <q-page class="index-page">
    <div class="page-inner">

      <!-- 页面标题 -->
      <div class="page-hero">
        <h1 class="hero-title">悬赏大厅</h1>
        <p class="hero-subtitle">发现机会，完成任务，赚取报酬</p>
      </div>

      <!-- 筛选栏 -->
      <div class="filters-bar">
        <div class="filter-tabs">
          <button
            v-for="tab in statusTabs"
            :key="tab.value"
            :class="['filter-tab', { active: activeStatus === tab.value }]"
            @click="switchTab(tab.value)"
          >
            {{ tab.label }}
          </button>
        </div>
        <div v-if="authStore.isLoggedIn()" class="create-btn-wrap">
          <q-btn unelevated color="primary" icon="add" label="发布悬赏" to="/bounty/create" />
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="bountyStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <!-- 悬赏列表 -->
      <div v-else-if="bountyStore.bounties.length" class="bounty-grid">
        <BountyCard
          v-for="(bounty, i) in bountyStore.bounties"
          :key="bounty.id"
          :bounty="bounty"
          :style="{ animationDelay: `${i * 80}ms` }"
          class="card-reveal"
        />
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">◈</div>
        <h3>暂无悬赏</h3>
        <p>目前没有符合条件的悬赏任务</p>
        <q-btn v-if="authStore.isLoggedIn()" unelevated color="primary" label="发布悬赏" to="/bounty/create" class="q-mt-md" />
      </div>

      <!-- 分页 -->
      <div v-if="bountyStore.total > pageSize" class="pagination">
        <q-pagination
          v-model="currentPage"
          :max="Math.ceil(bountyStore.total / pageSize)"
          :max-pages="7"
          boundary-numbers
          color="amber"
          @update:model-value="loadPage"
        />
      </div>

    </div>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useBountyStore } from 'src/stores/bounty'
import { useAuthStore } from 'src/stores/auth'
import BountyCard from 'src/components/bounty/BountyCard.vue'

const route = useRoute()
const router = useRouter()
const bountyStore = useBountyStore()
const authStore = useAuthStore()

const activeStatus = ref(route.query.status || 'PENDING')
const currentPage = ref(Number(route.query.page) || 1)
const pageSize = 12

const statusTabs = [
  { label: '全部', value: '' },
  { label: '待接单', value: 'PENDING' },
  { label: '进行中', value: 'IN_PROGRESS' },
  { label: '已完成', value: 'COMPLETED' },
  { label: '已取消', value: 'CANCELED' },
]

async function loadBounties() {
  bountyStore.resetState()
  await bountyStore.fetchBounties({ status: activeStatus.value, page: currentPage.value, pageSize })
}

function switchTab(status) {
  activeStatus.value = status
  currentPage.value = 1
  router.replace({ query: { status: status || undefined, page: undefined } })
  loadBounties()
}

function loadPage(page) {
  currentPage.value = page
  router.replace({ query: { ...route.query, page } })
  loadBounties()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

onMounted(loadBounties)
</script>

<style scoped lang="scss">
.index-page {
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

.filters-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 32px;
  flex-wrap: wrap;
}

.filter-tabs {
  display: flex;
  gap: 4px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 4px;
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
  white-space: nowrap;

  &:hover { color: var(--color-text-primary); background: var(--color-bg-elevated); }
  &.active {
    background: var(--color-accent-gold);
    color: #0b0d12;
    font-weight: 600;
  }
}

.bounty-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
  @media (max-width: 480px) {
    grid-template-columns: 1fr;
  }
}

.loading-area {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 240px;
}

.empty-state {
  text-align: center;
  padding: 80px 24px;
  .empty-icon {
    font-size: 4rem;
    color: var(--color-border);
    margin-bottom: 16px;
  }
  h3 {
    font-family: var(--font-display);
    font-size: 1.5rem;
    color: var(--color-text-muted);
    margin-bottom: 8px;
  }
  p { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 48px;
}
</style>
