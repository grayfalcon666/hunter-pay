<template>
  <q-layout view="lHh LpR lFf">

    <!-- 顶部导航栏 -->
    <q-header class="app-header">
      <q-toolbar class="toolbar-inner">

        <!-- Logo -->
        <router-link to="/" class="logo-link">
          <div class="logo-mark">
            <span class="logo-icon">◈</span>
            <span class="logo-text">Escrow</span>
          </div>
        </router-link>

        <q-space />

        <!-- 角色切换（桌面） -->
        <div v-if="authStore.isLoggedIn()" class="role-switch gt-sm">
          <button
            :class="['role-btn', { active: authStore.isPoster }]"
            @click="switchToEmployer"
          >
            <span class="role-dot poster-dot"></span>
            雇主
          </button>
          <button
            :class="['role-btn', { active: authStore.isHunter }]"
            @click="switchToHunter"
          >
            <span class="role-dot hunter-dot"></span>
            猎人
          </button>
        </div>

        <!-- 导航菜单（桌面） -->
        <div class="nav-tabs gt-sm">
          <q-tabs
            dense
            no-caps
            :active-color="authStore.isHunter ? 'accent-teal' : 'accent-gold'"
            :indicator-color="authStore.isHunter ? 'accent-teal' : 'accent-gold'"
            align="left"
            narrow-indicator
          >
            <!-- 雇主视图 -->
            <template v-if="authStore.isPoster">
              <q-route-tab to="/hunters" exact label="猎人招募" />
              <q-route-tab v-if="authStore.isLoggedIn()" to="/bounty/create" label="发布悬赏" />
              <q-route-tab v-if="authStore.isLoggedIn()" to="/my-bounties" label="我的悬赏" />
            </template>
            <!-- 猎人视图 -->
            <template v-else>
              <q-route-tab to="/" exact label="悬赏大厅" />
              <q-route-tab v-if="authStore.isLoggedIn()" to="/my-tasks" label="我的任务" />
            </template>
            <q-route-tab v-if="authStore.isLoggedIn()" to="/my/comments" label="我的评论" />
            <q-route-tab v-if="authStore.isLoggedIn()" to="/wallet" label="钱包" />
          </q-tabs>
        </div>

        <q-space />

        <!-- 用户区 -->
        <div v-if="authStore.isLoggedIn()" class="user-area">
          <!-- 角色标识 -->
          <span :class="['role-badge', authStore.isHunter ? 'role-badge--hunter' : 'role-badge--poster']">
            {{ authStore.isHunter ? '猎人' : '雇主' }}
          </span>
          <!-- 聊天按钮 -->
          <q-btn
            flat dense round
            icon="chat_bubble"
            class="chat-btn"
            @click="chatStore.openChat"
          >
            <q-badge
              v-if="chatStore.totalUnread > 0"
              floating color="negative"
              :label="chatStore.totalUnread > 99 ? '99+' : chatStore.totalUnread"
            />
          </q-btn>
          <router-link :to="`/profile/${authStore.username}`" class="user-name">
            {{ authStore.username }}
          </router-link>
          <q-btn flat dense round icon="logout" class="logout-btn" @click="handleLogout" />
        </div>
        <div v-else class="auth-btns">
          <q-btn flat label="登录" to="/login" class="auth-btn" />
          <q-btn unelevated label="注册" to="/register" color="primary" class="auth-btn--primary" />
        </div>

        <!-- 移动端菜单 -->
        <q-btn flat dense round icon="menu" class="lt-md" @click="toggleDrawer" />
      </q-toolbar>
    </q-header>

    <!-- 侧边抽屉（移动端） -->
    <q-drawer v-model="drawerOpen" side="right" overlay behavior="mobile" class="app-drawer">
      <div class="drawer-content">
        <div class="drawer-header">
          <span class="logo-icon">◈</span>
          <span class="logo-text">Escrow</span>
        </div>

        <q-separator color="border" class="q-my-sm" />

        <!-- 角色切换（移动端） -->
        <div v-if="authStore.isLoggedIn()" class="drawer-role-switch q-pa-sm">
          <button
            :class="['role-btn', { active: authStore.isPoster }]"
            @click="switchToEmployer"
          >
            <span class="role-dot poster-dot"></span>
            雇主
          </button>
          <button
            :class="['role-btn', { active: authStore.isHunter }]"
            @click="switchToHunter"
          >
            <span class="role-dot hunter-dot"></span>
            猎人
          </button>
        </div>

        <q-list padding>
          <!-- 雇主视图 -->
          <template v-if="authStore.isPoster">
            <q-item clickable v-ripple to="/hunters" @click="drawerOpen = false">
              <q-item-section avatar><q-icon name="people" /></q-item-section>
              <q-item-section>猎人招募</q-item-section>
            </q-item>
            <q-item v-if="authStore.isLoggedIn()" clickable v-ripple to="/bounty/create" @click="drawerOpen = false">
              <q-item-section avatar><q-icon name="add_circle" /></q-item-section>
              <q-item-section>发布悬赏</q-item-section>
            </q-item>
            <q-item v-if="authStore.isLoggedIn()" clickable v-ripple to="/my-bounties" @click="drawerOpen = false">
              <q-item-section avatar><q-icon name="work" /></q-item-section>
              <q-item-section>我的悬赏</q-item-section>
            </q-item>
          </template>
          <!-- 猎人视图 -->
          <template v-else>
            <q-item clickable v-ripple to="/" @click="drawerOpen = false">
              <q-item-section avatar><q-icon name="explore" /></q-item-section>
              <q-item-section>悬赏大厅</q-item-section>
            </q-item>
            <q-item v-if="authStore.isLoggedIn()" clickable v-ripple to="/my-tasks" @click="drawerOpen = false">
              <q-item-section avatar><q-icon name="task" /></q-item-section>
              <q-item-section>我的任务</q-item-section>
            </q-item>
          </template>
          <q-item v-if="authStore.isLoggedIn()" clickable v-ripple to="/my/comments" @click="drawerOpen = false">
            <q-item-section avatar><q-icon name="comment" /></q-item-section>
            <q-item-section>我的评论</q-item-section>
          </q-item>
          <q-item v-if="authStore.isLoggedIn()" clickable v-ripple to="/wallet" @click="drawerOpen = false">
            <q-item-section avatar><q-icon name="account_balance_wallet" /></q-item-section>
            <q-item-section>钱包</q-item-section>
          </q-item>
          <q-item v-if="authStore.isLoggedIn()" clickable v-ripple :to="`/profile/${authStore.username}`" @click="drawerOpen = false">
            <q-item-section avatar><q-icon name="person" /></q-item-section>
            <q-item-section>个人资料</q-item-section>
          </q-item>
        </q-list>

        <q-separator color="border" class="q-my-sm" />

        <div v-if="authStore.isLoggedIn()" class="q-pa-md">
          <q-btn flat fullwidth label="退出登录" icon="logout" @click="handleLogout" />
        </div>
        <div v-else class="q-pa-md">
          <q-btn flat fullwidth label="登录" to="/login" class="q-mb-sm" />
          <q-btn unelevated fullwidth label="注册" to="/register" color="primary" />
        </div>
      </div>
    </q-drawer>

    <!-- 页面容器 -->
    <q-page-container>
      <router-view />
    </q-page-container>

    <!-- 全局聊天抽屉 -->
    <ChatDrawer />

  </q-layout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from 'src/stores/auth'
import { useChatStore } from 'src/stores/chat'
import ChatDrawer from 'src/components/chat/ChatDrawer.vue'

const authStore = useAuthStore()
const chatStore = useChatStore()
const router = useRouter()
const drawerOpen = ref(false)

function toggleDrawer() {
  drawerOpen.value = !drawerOpen.value
}

function switchToEmployer() {
  const wasHunter = authStore.isHunter
  authStore.setRole('EMPLOYER')
  drawerOpen.value = false
  if (wasHunter) {
    router.push('/hunters')
  }
}

function switchToHunter() {
  const wasPoster = authStore.isPoster
  authStore.setRole('HUNTER')
  drawerOpen.value = false
  if (wasPoster) {
    router.push('/')
  }
}

function handleLogout() {
  authStore.logout()
  chatStore.cleanup()
  drawerOpen.value = false
  router.push('/')
}

onMounted(() => {
  chatStore.init()
})
</script>

<style scoped lang="scss">
.app-header {
  background: rgba(11, 13, 18, 0.88) !important;
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid var(--color-border);
  box-shadow: 0 1px 0 0 var(--color-accent-gold), 0 4px 24px rgba(0,0,0,0.4);
}

.toolbar-inner {
  max-width: 1280px;
  margin: 0 auto;
  width: 100%;
  padding: 0 24px;
}

.logo-link { text-decoration: none; }

.logo-mark {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo-icon {
  font-size: 1.4rem;
  color: var(--color-accent-gold);
  line-height: 1;
}

.logo-text {
  font-family: var(--font-display);
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

// 角色切换按钮
.role-switch {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 4px;
}

.role-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: 0.8rem;
  font-weight: 500;
  padding: 6px 12px;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover {
    color: var(--color-text-primary);
    background: var(--color-bg-elevated);
  }

  &.active {
    font-weight: 600;
  }
}

.role-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.poster-dot {
  background: var(--color-accent-gold);
  box-shadow: 0 0 6px var(--color-glow-gold);
}

.hunter-dot {
  background: var(--color-accent-teal);
  box-shadow: 0 0 6px var(--color-glow-teal);
}

.role-btn.active.poster { color: var(--color-accent-gold); }
.role-btn.active.hunter { color: var(--color-accent-teal); }

// 角色标识
.role-badge {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 4px;
  letter-spacing: 0.05em;
  text-transform: uppercase;

  &--poster {
    color: var(--color-accent-gold);
    background: var(--color-glow-gold);
    border: 1px solid rgba(201, 168, 76, 0.3);
  }

  &--hunter {
    color: var(--color-accent-teal);
    background: var(--color-glow-teal);
    border: 1px solid rgba(45, 212, 191, 0.3);
  }
}

// q-tabs 样式覆盖
.nav-tabs {
  :deep(.q-tab) {
    font-family: var(--font-body);
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--color-text-muted);
    padding: 0 14px;
    min-height: 52px;
    transition: color 0.15s;

    &:hover {
      color: var(--color-text-primary);
    }

    &.q-tab--active {
      color: var(--color-accent-gold);
    }
  }

  :deep(.q-tabs__content) {
    gap: 4px;
  }

  :deep(.q-tab__indicator) {
    height: 2px;
    border-radius: 1px;
  }
}

.user-area {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chat-btn {
  color: var(--color-text-muted);
  &:hover { color: var(--color-accent-gold); }
}

.user-name {
  font-family: var(--font-body);
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--color-text-primary);
  text-decoration: none;
  &:hover { color: var(--color-accent-gold); }
}

.logout-btn {
  color: var(--color-text-muted);
  &:hover { color: var(--color-accent-red); }
}

.auth-btns {
  display: flex;
  align-items: center;
  gap: 8px;
}

.auth-btn {
  color: var(--color-text-muted);
  font-size: 0.9rem;
  &:hover { color: var(--color-text-primary); }
}

.auth-btn--primary {
  font-size: 0.85rem;
  font-weight: 600;
}

// 侧边栏
.app-drawer {
  background: var(--color-bg-secondary) !important;
}

.drawer-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px 0;
}

.drawer-role-switch {
  display: flex;
  gap: 4px;
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  margin: 0 16px;

  .role-btn {
    flex: 1;
  }
}

.drawer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px 8px;
  .logo-icon { font-size: 1.4rem; color: var(--color-accent-gold); }
  .logo-text {
    font-family: var(--font-display);
    font-size: 1.3rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
}
</style>
