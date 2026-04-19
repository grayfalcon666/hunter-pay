<template>
  <q-page class="edit-profile-page">
    <div class="page-inner">
      <div class="edit-card-wrapper">
        <q-card class="edit-card">
          <q-card-section class="edit-header">
            <h2 class="edit-title">编辑资料</h2>
            <p class="edit-hint">更新您的个人资料信息</p>
          </q-card-section>

          <q-separator color="border" />

          <q-card-section>
            <q-form @submit.prevent="handleSubmit" class="edit-form">
              <div class="form-field">
                <label class="field-label">真实姓名</label>
                <q-input
                  v-model="form.full_name"
                  outlined
                  dense
                  placeholder="输入您的真实姓名"
                  :disable="submitting"
                />
              </div>

              <div class="form-field">
                <label class="field-label">个人简介</label>
                <q-input
                  v-model="form.bio"
                  outlined
                  dense
                  type="textarea"
                  :rows="4"
                  placeholder="介绍一下自己..."
                  :disable="submitting"
                />
              </div>

              <div class="form-actions">
                <q-btn
                  flat
                  label="取消"
                  @click="handleCancel"
                  :disable="submitting"
                />
                <q-btn
                  unelevated
                  label="保存"
                  color="primary"
                  type="submit"
                  :loading="submitting"
                />
              </div>
            </q-form>
          </q-card-section>
        </q-card>
      </div>
    </div>
  </q-page>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useProfileStore } from 'src/stores/profile'
import { useAuthStore } from 'src/stores/auth'

const router = useRouter()
const $q = useQuasar()
const profileStore = useProfileStore()
const authStore = useAuthStore()

const form = ref({
  full_name: '',
  bio: '',
})

const submitting = ref(false)

onMounted(async () => {
  await profileStore.fetchProfile(authStore.username)
  const p = profileStore.profile
  if (p) {
    form.value.full_name = p.full_name || ''
    form.value.bio = p.bio || ''
  }
})

async function handleSubmit() {
  submitting.value = true
  try {
    await profileStore.updateProfile(authStore.username, {
      full_name: form.value.full_name,
      bio: form.value.bio,
    })
    $q.notify({ type: 'positive', message: '资料更新成功' })
    router.push(`/profile/${authStore.username}`)
  } catch (e) {
    $q.notify({ type: 'negative', message: e.message || '更新失败' })
  } finally {
    submitting.value = false
  }
}

function handleCancel() {
  router.push(`/profile/${authStore.username}`)
}
</script>

<style scoped lang="scss">
.edit-profile-page {
  background: var(--color-bg-primary);
  min-height: 100vh;
}

.page-inner {
  max-width: 640px;
  margin: 0 auto;
  padding: 48px 24px;
}

.edit-card {
  background: var(--color-bg-secondary) !important;
  border: 1px solid var(--color-border) !important;
  border-radius: var(--radius-card) !important;
}

.edit-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.edit-title {
  font-family: var(--font-display);
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0;
}

.edit-hint {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  margin: 0;
}

.edit-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--color-text-muted);
  letter-spacing: 0.04em;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}
</style>
