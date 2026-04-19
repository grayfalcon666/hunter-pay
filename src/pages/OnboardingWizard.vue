<template>
  <q-page class="q-pa-md">
    <div class="row justify-center q-mb-lg">
      <h5 class="q-mb-md">欢迎！让我们完成账户设置</h5>
    </div>

    <q-stepper
      v-model="step"
      color="primary"
      animated
      header-nav
    >
      <q-step
        :step="1"
        title="完善个人资料"
        icon="person"
        :name="1"
      >
        <div class="q-pa-md">
          <p class="q-mb-md">请填写您的基本信息，帮助雇主更好地了解您</p>

          <q-form @submit="submitProfile" class="q-gutter-md">
            <q-input
              v-model="profile.expectedSalaryMin"
              label="期望薪资最低 (元)"
              type="number"
              outlined
            />

            <q-input
              v-model="profile.expectedSalaryMax"
              label="期望薪资最高 (元)"
              type="number"
              outlined
            />

            <q-input
              v-model="profile.workLocation"
              label="工作地点"
              placeholder="如：北京、上海、远程"
              outlined
            />

            <q-select
              v-model="profile.experienceLevel"
              :options="experienceLevels"
              label="经验水平"
              outlined
              emit-value
              map-options
            />

            <q-input
              v-model="profile.bio"
              label="个人简介"
              type="textarea"
              outlined
              counter
            />

            <div class="row justify-end q-mt-md">
              <q-btn type="submit" label="下一步" color="primary" />
            </div>
          </q-form>
        </div>
      </q-step>

      <q-step
        :step="2"
        title="创建第一个悬赏任务"
        icon="assignment"
        :name="2"
      >
        <div class="q-pa-md">
          <p class="q-mb-md">您可以作为雇主发布悬赏任务，或作为猎人接取任务</p>

          <div class="row q-col-gutter-md">
            <div class="col-6">
              <q-card>
                <q-card-section>
                  <div class="text-h6">作为雇主</div>
                  <div class="text-subtitle2">发布悬赏任务</div>
                </q-card-section>
                <q-card-actions align="right">
                  <q-btn label="发布悬赏" color="primary" @click="navigateToCreateBounty" />
                </q-card-actions>
              </q-card>
            </div>
            <div class="col-6">
              <q-card>
                <q-card-section>
                  <div class="text-h6">作为猎人</div>
                  <div class="text-subtitle2">浏览悬赏任务</div>
                </q-card-section>
                <q-card-actions align="right">
                  <q-btn label="浏览任务" color="primary" @click="navigateToBountyList" />
                </q-card-actions>
              </q-card>
            </div>
          </div>

          <div class="row justify-center q-mt-md">
            <q-btn label="跳过" color="grey" flat @click="skipOnboarding" />
          </div>
        </div>
      </q-step>

      <q-step
        :step="3"
        title="账户充值"
        icon="wallet"
        :name="3"
      >
        <div class="q-pa-md">
          <p class="q-mb-md">为账户充值以便接取悬赏任务</p>

          <q-card class="q-mb-md">
            <q-card-section>
              <div class="text-h6">当前余额</div>
              <div class="text-h4 text-primary">{{ balance }} 元</div>
            </q-card-section>
          </q-card>

          <q-btn
            color="primary"
            label="立即充值"
            class="full-width"
            @click="navigateToWallet"
          />

          <div class="row justify-center q-mt-md">
            <q-btn label="稍后充值" color="grey" flat @click="skipOnboarding" />
          </div>
        </div>
      </q-step>

      <q-step
        :step="4"
        title="完成"
        icon="check_circle"
        :name="4"
      >
        <div class="q-pa-md text-center">
          <q-icon name="check_circle" color="positive" size="64px" />
          <h5 class="q-mt-md">欢迎加入平台!</h5>
          <p class="text-grey-7">您的账户已完全设置完成</p>

          <q-btn
            color="primary"
            label="开始使用"
            class="q-mt-md"
            @click="navigateToHome"
          />
        </div>
      </q-step>
    </q-stepper>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProfileStore } from '@/stores/profile'
import { getUserBalance } from '@/api/account'

const router = useRouter()
const profileStore = useProfileStore()

const step = ref(1)
const profile = ref({
  expectedSalaryMin: '',
  expectedSalaryMax: '',
  workLocation: '',
  experienceLevel: 'ENTRY',
  bio: '',
  avatarUrl: ''
})
const balance = ref(0)

const experienceLevels = [
  { label: '入门', value: 'ENTRY' },
  { label: '初级', value: 'JUNIOR' },
  { label: '中级', value: 'MID' },
  { label: '高级', value: 'SENIOR' },
  { label: '专家', value: 'EXPERT' }
]

onMounted(async () => {
  // 检查是否已初始化过
  const hasOnboarded = localStorage.getItem('hasOnboarded')
  if (hasOnboarded === 'true') {
    router.push('/')
    return
  }

  // 获取用户余额
  try {
    const response = await getUserBalance()
    balance.value = response.balance / 100 // 转换为元
  } catch (error) {
    console.error('获取余额失败:', error)
  }
})

async function submitProfile() {
  try {
    await profileStore.createProfile(profile.value)
    step.value = 2
  } catch (error) {
    console.error('创建资料失败:', error)
  }
}

function navigateToCreateBounty() {
  router.push('/bounties/create')
}

function navigateToBountyList() {
  router.push('/bounties')
}

function navigateToWallet() {
  router.push('/wallet')
}

function navigateToHome() {
  localStorage.setItem('hasOnboarded', 'true')
  router.push('/')
}

function skipOnboarding() {
  localStorage.setItem('hasOnboarded', 'true')
  router.push('/')
}
</script>

<style scoped>
.full-width {
  width: 100%;
}
</style>
