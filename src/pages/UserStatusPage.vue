<template>
  <q-page class="q-pa-md flex flex-center">
    <div class="column items-center q-pa-xl" style="max-width: 400px;">
      <h5 class="q-mb-md">正在初始化账户...</h5>

      <div class="q-mb-lg">
        <q-spinner-dots color="primary" size="50px" />
      </div>

      <q-card class="q-mb-md" style="min-width: 300px;">
        <q-card-section>
          <div class="text-subtitle2 text-grey-7">用户名</div>
          <div class="text-h6">{{ username }}</div>
        </q-card-section>
      </q-card>

      <q-card v-if="status === 'INITIALIZED'" class="bg-positive text-white q-mb-md">
        <q-card-section>
          <div class="text-h6">初始化完成！</div>
          <div class="text-subtitle2">您的账户已准备就绪</div>
        </q-card-section>
      </q-card>

      <q-card v-if="status === 'FAILED'" class="bg-negative text-white q-mb-md">
        <q-card-section>
          <div class="text-h6">初始化失败</div>
          <div class="text-subtitle2">{{ errorMessage || '请稍后重试' }}</div>
          <q-btn
            color="white"
            label="重试"
            class="q-mt-sm"
            @click="checkStatus"
          />
        </q-card-section>
      </q-card>

      <q-card v-if="status === 'PARTIALLY_INITIALIZED'" class="bg-orange text-white q-mb-md">
        <q-card-section>
          <div class="text-h6">部分初始化完成</div>
          <div class="text-subtitle2">您可以继续使用，但建议完善账户</div>
          <q-btn
            color="white"
            label="继续"
            class="q-mt-sm"
            @click="navigateToHome"
          />
        </q-card-section>
      </q-card>

      <div class="text-caption text-grey-6 q-mt-md">
        如长时间未响应，请<a href="/login" class="text-primary">返回登录</a>
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { getUserStatus } from 'src/api/auth'

const router = useRouter()
const $q = useQuasar()

const username = ref('')
const status = ref('PROFILE_PENDING')
const errorMessage = ref('')
let pollingInterval = null

// 初始化状态
onMounted(() => {
  // 从 localStorage 获取用户名
  const storedUsername = localStorage.getItem('username')
  if (!storedUsername) {
    // 没有用户名，跳转到登录页
    router.push('/login')
    return
  }
  username.value = storedUsername

  // 开始轮询状态
  checkStatus()
  pollingInterval = setInterval(checkStatus, 3000)
})

// 检查用户状态
async function checkStatus() {
  try {
    const response = await getUserStatus(username.value)
    status.value = response.status || 'PROFILE_PENDING'
    errorMessage.value = response.failed_reason || ''

    // 根据状态跳转
    if (status.value === 'INITIALIZED' || status.value === 'PARTIALLY_INITIALIZED') {
      clearInterval(pollingInterval)
      // 跳转到引导页面或首页
      setTimeout(() => {
        router.push('/onboarding')
      }, 1000)
    } else if (status.value === 'FAILED') {
      // 显示错误提示
      $q.notify({
        type: 'negative',
        message: '账户初始化失败，请重试',
        position: 'top'
      })
    }
  } catch (error) {
    console.error('获取用户状态失败:', error)
    // 继续轮询
  }
}

// 导航到首页
function navigateToHome() {
  router.push('/')
}

// 组件卸载时清除轮询
onUnmounted(() => {
  if (pollingInterval) {
    clearInterval(pollingInterval)
  }
})
</script>

<style scoped>
.column {
  align-items: center;
}
</style>
