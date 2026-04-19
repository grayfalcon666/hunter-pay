<template>
  <q-page class="my-tasks-page">
    <div class="page-inner">

      <!-- 页面标题 -->
      <div class="page-hero">
        <h1 class="hero-title">我的任务</h1>
        <p class="hero-subtitle">管理您的接单任务与收到的邀请</p>
      </div>

      <!-- Tab 切换 -->
      <div class="tabs-bar">
        <button
          :class="['tab-btn', { active: activeTab === 'invitations' }]"
          @click="switchTab('invitations')"
        >
          <q-icon name="mail" />
          收到邀请
          <span v-if="pendingCount > 0" class="tab-badge">{{ pendingCount }}</span>
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'applications' }]"
          @click="switchTab('applications')"
        >
          <q-icon name="send" />
          我的申请
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'tasks' }]"
          @click="switchTab('tasks')"
        >
          <q-icon name="task" />
          已接受任务
        </button>
      </div>

      <!-- 收到邀请 -->
      <div v-if="activeTab === 'invitations'">
        <div v-if="loading" class="loading-area">
          <q-spinner-dots color="teal" size="40px" />
        </div>

        <div v-else-if="invitations.length === 0" class="empty-state">
          <div class="empty-icon">◈</div>
          <h3>暂无邀请</h3>
          <p>猎人可以在悬赏大厅主动申请任务</p>
        </div>

        <div v-else class="invitation-list">
          <div
            v-for="(inv, i) in invitations"
            :key="inv.id"
            class="invitation-card card-reveal"
            :style="{ animationDelay: `${i * 80}ms` }"
          >
            <div class="inv-header">
              <div class="poster-info">
                <div class="poster-avatar">{{ (inv.poster_username || '?')[0].toUpperCase() }}</div>
                <div>
                  <div class="poster-name">{{ inv.poster_username }}</div>
                  <div class="inv-time">{{ formatTime(inv.created_at) }}</div>
                </div>
              </div>
              <span :class="['status-badge', `status-${inv.status.toLowerCase()}`]">
                {{ statusLabel(inv.status) }}
              </span>
            </div>

            <div v-if="inv.bounty" class="bounty-preview">
              <h4 class="bounty-title">{{ inv.bounty.title }}</h4>
              <div class="bounty-meta">
                <span class="bounty-amount">{{ formatYuan(inv.bounty.reward_amount) }}</span>
                <span class="bounty-status">{{ inv.bounty.status }}</span>
              </div>
            </div>

            <div v-if="inv.status === 'PENDING'" class="inv-actions">
              <q-btn
                unelevated
                color="teal"
                label="接受"
                class="action-btn"
                @click="handleRespond(inv.id, true)"
                :loading="respondingId === inv.id"
              />
              <q-btn
                flat
                color="grey"
                label="拒绝"
                class="action-btn"
                @click="handleRespond(inv.id, false)"
                :loading="respondingId === inv.id"
              />
              <q-btn
                flat
                dense
                icon="chat"
                label="发消息"
                class="msg-btn"
                @click="openChat(inv.poster_username)"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- 我的申请 -->
      <div v-if="activeTab === 'applications'">
        <div v-if="appsLoading" class="loading-area">
          <q-spinner-dots color="teal" size="40px" />
        </div>

        <div v-else-if="myApplications.length === 0" class="empty-state">
          <div class="empty-icon">◈</div>
          <h3>暂无申请记录</h3>
          <p>去悬赏大厅主动申请任务吧</p>
          <q-btn unelevated color="teal" label="浏览悬赏大厅" to="/" class="q-mt-md" />
        </div>

        <div v-else class="invitation-list">
          <div
            v-for="(app, i) in myApplications"
            :key="app.id"
            class="invitation-card card-reveal"
            :style="{ animationDelay: `${i * 80}ms` }"
          >
            <div class="inv-header">
              <div class="poster-info">
                <div class="poster-avatar">{{ (app.bounty?.employerUsername || app.bounty?.employer_username || '?')[0].toUpperCase() }}</div>
                <div>
                  <div class="poster-name">{{ app.bounty?.employerUsername || app.bounty?.employer_username || '未知' }}</div>
                  <div class="inv-time">{{ formatTime(app.created_at) }}</div>
                </div>
              </div>
              <span :class="['status-badge', `status-${app.status.toLowerCase()}`]">
                {{ appStatusLabel(app.status) }}
              </span>
            </div>

            <div v-if="app.bounty" class="bounty-preview">
              <h4 class="bounty-title">{{ app.bounty.title }}</h4>
              <div class="bounty-meta">
                <span class="bounty-amount">{{ formatYuan(app.bounty.rewardAmount ?? app.bounty.reward_amount) }}</span>
                <span class="bounty-status">{{ app.bounty.status }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 已接受任务 -->
      <div v-if="activeTab === 'tasks'">
        <div v-if="tasksLoading" class="loading-area">
          <q-spinner-dots color="teal" size="40px" />
        </div>

        <div v-else-if="tasks.length === 0" class="empty-state">
          <div class="empty-icon">◈</div>
          <h3>暂无进行中的任务</h3>
          <p>去悬赏大厅发现机会，或接受邀请开始接单</p>
          <q-btn unelevated color="teal" label="浏览悬赏大厅" to="/" class="q-mt-md" />
        </div>

        <div v-else class="tasks-grid">
          <div
            v-for="(task, i) in tasks"
            :key="task.id"
            :class="['task-card', 'card-reveal', { 'task-completed': task.status === 'COMPLETED' }]"
            :style="{ animationDelay: `${i * 80}ms` }"
            @click="goToBounty(task.id)"
          >
            <div class="task-header">
              <h4 class="task-title">{{ task.title }}</h4>
              <span :class="['status-badge', `status-${task.status.toLowerCase()}`]">
                {{ task.status === 'COMPLETED' ? '已完成' : task.status }}
              </span>
            </div>
            <div class="task-meta">
              <span class="task-amount">{{ formatYuan(task.rewardAmount ?? task.reward_amount) }}</span>
              <span class="task-employer">雇主: {{ task.employerUsername || task.employer_username }}</span>
            </div>
            <div v-if="task.submission_text" class="task-submission">
              <span class="submission-label">已提交:</span>
              <p class="submission-text">{{ task.submission_text }}</p>
            </div>
          </div>
        </div>
      </div>

    </div>
  </q-page>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBountyStore } from 'src/stores/bounty'
import { useChatStore } from 'src/stores/chat'
import { useAuthStore } from 'src/stores/auth'
import { getMyInvitations, respondToInvitation, getMyApplications } from 'src/api/invitation'

const router = useRouter()
const bountyStore = useBountyStore()
const chatStore = useChatStore()
const authStore = useAuthStore()

const activeTab = ref('invitations')
const loading = ref(false)
const tasksLoading = ref(false)
const appsLoading = ref(false)
const invitations = ref([])
const tasks = ref([])
const myApplications = ref([])
const respondingId = ref(null)

const pendingCount = computed(() =>
  invitations.value.filter(inv => inv.status === 'PENDING').length
)

function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'invitations') {
    loadInvitations()
  } else if (tab === 'applications') {
    loadMyApplications()
  } else {
    loadTasks()
  }
}

async function loadInvitations() {
  loading.value = true
  try {
    const data = await getMyInvitations({ status: '', pageSize: 50 })
    invitations.value = data.invitations || []
  } catch (e) {
    console.error('loadInvitations error:', e)
  } finally {
    loading.value = false
  }
}

async function loadTasks() {
  tasksLoading.value = true
  try {
    // 加载进行中的悬赏
    await bountyStore.fetchBounties({ status: 'IN_PROGRESS', pageSize: 50 })
    const inProgressTasks = bountyStore.bounties || []
    // 加载已完成的悬赏
    await bountyStore.fetchBounties({ status: 'COMPLETED', pageSize: 50 })
    const completedTasks = bountyStore.bounties || []
    // 合并并按创建时间倒序
    const allTasks = [...inProgressTasks, ...completedTasks]
    allTasks.sort((a, b) => new Date(b.created_at || b.createdAt) - new Date(a.created_at || a.createdAt))
    // 过滤：只显示当前用户作为猎人接单的任务，不显示自己作为雇主发布的任务
    tasks.value = allTasks.filter(t =>
      (t.employerUsername || t.employer_username) !== authStore.username
    )
  } catch (e) {
    console.error('loadTasks error:', e)
  } finally {
    tasksLoading.value = false
  }
}

async function loadMyApplications() {
  appsLoading.value = true
  try {
    const data = await getMyApplications({ status: '', pageSize: 50 })
    myApplications.value = data.applications || []
  } catch (e) {
    console.error('loadMyApplications error:', e)
  } finally {
    appsLoading.value = false
  }
}

async function handleRespond(invitationId, accept) {
  respondingId.value = invitationId
  try {
    await respondToInvitation(invitationId, accept)
    await loadInvitations()
  } catch (e) {
    console.error('respond error:', e)
  } finally {
    respondingId.value = null
  }
}

function openChat(username) {
  chatStore.startConversation(username)
}

function goToBounty(id) {
  router.push(`/bounty/${id}`)
}

function formatYuan(cents) {
  const n = typeof cents === 'string' ? parseInt(cents, 10) : cents
  if (isNaN(n)) return '¥ 0.00'
  return `¥ ${(n / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`
}

function formatTime(timestamp) {
  if (!timestamp) return ''
  const d = new Date(timestamp * 1000)
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function statusLabel(status) {
  const map = { PENDING: '待响应', ACCEPTED: '已接受', DECLINED: '已拒绝' }
  return map[status] || status
}

function appStatusLabel(status) {
  const map = { APPLIED: '申请中', CONFIRMED: '已确认', REJECTED: '已拒绝' }
  return map[status] || status
}

onMounted(() => {
  loadInvitations()
})
</script>

<style scoped lang="scss">
.my-tasks-page {
  min-height: 100vh;
  background: var(--color-bg-primary);
}

.page-inner {
  max-width: 900px;
  margin: 0 auto;
  padding: 48px 24px;
}

.page-hero {
  margin-bottom: 40px;
  text-align: center;
}

.hero-title {
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3rem);
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

.tabs-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 32px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 6px;
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: 0.9rem;
  font-weight: 500;
  padding: 12px 20px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover { color: var(--color-text-primary); background: var(--color-bg-elevated); }
  &.active {
    background: var(--color-accent-teal);
    color: #0b0d12;
    font-weight: 600;
  }
}

.tab-badge {
  background: var(--color-accent-red);
  color: white;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
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

.invitation-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.invitation-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  padding: 20px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    border-color: var(--color-accent-teal);
  }
}

.inv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.poster-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.poster-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-accent-teal);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 1rem;
  font-weight: 700;
  color: var(--color-accent-teal);
}

.poster-name { font-weight: 600; font-size: 0.95rem; }
.inv-time { font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono); }

.status-badge {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 4px;
  letter-spacing: 0.05em;
  text-transform: uppercase;

  &.status-pending { background: rgba(251, 191, 36, 0.15); color: var(--color-accent-amber); border: 1px solid rgba(251, 191, 36, 0.3); }
  &.status-accepted { background: rgba(45, 212, 191, 0.15); color: var(--color-accent-teal); border: 1px solid rgba(45, 212, 191, 0.3); }
  &.status-declined { background: rgba(248, 113, 113, 0.15); color: var(--color-accent-red); border: 1px solid rgba(248, 113, 113, 0.3); }
}

.bounty-preview {
  background: var(--color-bg-elevated);
  border-radius: 8px;
  padding: 14px;
  margin-bottom: 16px;
}

.bounty-title { font-size: 1rem; font-weight: 600; margin-bottom: 8px; color: var(--color-text-primary); }

.bounty-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bounty-amount { font-family: var(--font-mono); font-weight: 700; color: var(--color-accent-gold); }
.bounty-status { font-size: 0.75rem; color: var(--color-text-muted); font-family: var(--font-mono); text-transform: uppercase; }

.inv-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  min-width: 80px;
  font-weight: 600;
}

.msg-btn {
  margin-left: auto;
  color: var(--color-text-muted);
  &:hover { color: var(--color-accent-teal); }
}

.tasks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.task-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-card);
  padding: 20px;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s ease;

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    border-color: var(--color-accent-teal);
  }

  &.task-completed {
    opacity: 0.7;
    border-color: var(--color-accent-green);

    .task-title {
      text-decoration: line-through;
      color: var(--color-text-muted);
    }
  }
}

.task-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.task-title { font-size: 1rem; font-weight: 600; color: var(--color-text-primary); margin: 0; }

.task-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.task-amount { font-family: var(--font-mono); font-weight: 700; color: var(--color-accent-gold); }
.task-employer { font-size: 0.8rem; color: var(--color-text-muted); }

.task-submission {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.submission-label { font-size: 0.75rem; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
.submission-text { font-size: 0.85rem; color: var(--color-text-muted); margin-top: 4px; line-height: 1.4; }
</style>
