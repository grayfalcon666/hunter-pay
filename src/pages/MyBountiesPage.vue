<template>
  <q-page class="my-bounties-page">
    <div class="page-inner">
      <div class="page-header">
        <h1 class="page-title">我的悬赏</h1>
        <p class="page-subtitle">作为雇主发布的悬赏任务</p>
      </div>

      <div v-if="bountyStore.loading" class="loading-area">
        <q-spinner-dots color="amber" size="40px" />
      </div>

      <div v-else-if="myBounties.length" class="bounty-list">
        <BountyCard
          v-for="(bounty, i) in myBounties"
          :key="bounty.id"
          :bounty="bounty"
          :style="{ animationDelay: `${i * 80}ms` }"
          class="card-reveal"
        />
      </div>

      <div v-else class="empty-state">
        <div class="empty-icon">◈</div>
        <h3>暂无发布</h3>
        <p>您还没有发布过悬赏任务</p>
        <q-btn unelevated color="primary" label="发布悬赏" to="/bounty/create" class="q-mt-md" />
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useBountyStore } from 'src/stores/bounty'
import { useAuthStore } from 'src/stores/auth'
import BountyCard from 'src/components/bounty/BountyCard.vue'

const bountyStore = useBountyStore()
const authStore = useAuthStore()

const myBounties = computed(() =>
  bountyStore.bounties.filter(b => (b.employerUsername ?? b.employer_username) === authStore.username)
)

onMounted(() => { bountyStore.resetState(); bountyStore.fetchBounties({ pageSize: 50 }) })
</script>

<style scoped lang="scss">
.my-bounties-page { background: var(--color-bg-primary); min-height: 100vh; }
.page-inner { max-width: 1280px; margin: 0 auto; padding: 48px 24px; }
.page-title { font-family: var(--font-display); font-size: 2rem; font-weight: 700; margin-bottom: 6px; }
.page-subtitle { color: var(--color-text-muted); font-size: 0.9rem; margin-bottom: 32px; }
.loading-area { display: flex; justify-content: center; padding: 80px; }
.bounty-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 20px; }
.empty-state {
  text-align: center; padding: 80px 24px;
  .empty-icon { font-size: 4rem; color: var(--color-border); margin-bottom: 16px; }
  h3 { font-family: var(--font-display); font-size: 1.5rem; color: var(--color-text-muted); margin-bottom: 8px; }
  p { color: var(--color-text-muted); font-size: 0.9rem; margin: 0; }
}
</style>
