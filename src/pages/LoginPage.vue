<template>
  <q-page class="auth-page">
    <div class="auth-container">
      <!-- 装饰背景 -->
      <div class="auth-bg">
        <div class="bg-glow" />
        <div class="bg-grid" />
      </div>

      <div class="auth-card">
        <!-- Logo -->
        <div class="auth-logo">
          <span class="logo-icon">◈</span>
          <span class="logo-text">Escrow</span>
        </div>

        <h2 class="auth-title">欢迎回来</h2>
        <p class="auth-subtitle">登录到您的账户</p>

        <q-form @submit="handleLogin" class="auth-form">

          <q-input
            v-model="form.username"
            label="用户名"
            outlined
            :rules="[val => !!val || '请输入用户名']"
            class="q-mb-md"
          >
            <template #prepend>
              <q-icon name="person" color="grey-6" />
            </template>
          </q-input>

          <q-input
            v-model="form.password"
            label="密码"
            :type="showPwd ? 'text' : 'password'"
            outlined
            :rules="[val => !!val || '请输入密码']"
            class="q-mb-lg"
          >
            <template #prepend>
              <q-icon name="lock" color="grey-6" />
            </template>
            <template #append>
              <q-icon
                :name="showPwd ? 'visibility_off' : 'visibility'"
                class="cursor-pointer"
                color="grey-6"
                @click="showPwd = !showPwd"
              />
            </template>
          </q-input>

          <q-btn
            type="submit"
            :loading="loading"
            unelevated
            color="primary"
            label="登录"
            class="submit-btn"
            no-caps
          />
        </q-form>

        <p class="auth-footer">
          还没有账户？
          <router-link to="/register" class="auth-link">立即注册</router-link>
        </p>
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { useAuthStore } from 'src/stores/auth'
import { login } from 'src/api/auth'

const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()
const $q = useQuasar()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const showPwd = ref(false)

async function handleLogin() {
  loading.value = true
  try {
    const data = await login(form.value)
    const token = data.access_token
    authStore.setAuth(token, form.value.username)
    $q.notify({ type: 'positive', message: '登录成功' })
    const redirect = route.query.redirect || '/'
    router.push(redirect)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '登录失败' })
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-primary);
  position: relative;
  overflow: hidden;
}

.auth-container {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 440px;
  padding: 24px;
}

.auth-bg {
  position: fixed;
  inset: 0;
  pointer-events: none;
  .bg-glow {
    position: absolute;
    top: -20%;
    left: 50%;
    transform: translateX(-50%);
    width: 600px;
    height: 600px;
    background: radial-gradient(ellipse at center, rgba(201,168,76,0.06) 0%, transparent 70%);
    border-radius: 50%;
  }
  .bg-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(42,47,61,0.3) 1px, transparent 1px),
      linear-gradient(90deg, rgba(42,47,61,0.3) 1px, transparent 1px);
    background-size: 48px 48px;
  }
}

.auth-card {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  padding: 48px 40px;
  position: relative;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: linear-gradient(90deg, transparent, var(--color-accent-gold), transparent);
  }
}

.auth-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 32px;
  .logo-icon { font-size: 1.6rem; color: var(--color-accent-gold); }
  .logo-text {
    font-family: var(--font-display);
    font-size: 1.5rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--color-text-primary);
  }
}

.auth-title {
  font-family: var(--font-display);
  font-size: 1.8rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 6px;
}

.auth-subtitle {
  text-align: center;
  color: var(--color-text-muted);
  font-size: 0.9rem;
  margin-bottom: 32px;
}

.auth-form { margin-bottom: 24px; }

.submit-btn {
  width: 100%;
  height: 48px;
  font-size: 1rem;
  font-weight: 600;
  border-radius: var(--radius-btn);
}

.auth-footer {
  text-align: center;
  color: var(--color-text-muted);
  font-size: 0.875rem;
  margin: 0;
}

.auth-link {
  color: var(--color-accent-gold);
  text-decoration: none;
  font-weight: 500;
  &:hover { text-decoration: underline; }
}
</style>
