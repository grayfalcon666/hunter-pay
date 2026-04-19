<template>
  <div class="chat-input-area">
    <q-input
      v-model="inputText"
      dense outlined
      placeholder="输入消息..."
      class="chat-input"
      :maxlength="2000"
      @keydown.enter.exact.prevent="handleSend"
    >
      <template #append>
        <q-btn
          flat dense round icon="send"
          color="primary"
          :disable="!inputText.trim()"
          @click="handleSend"
        />
      </template>
    </q-input>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['send'])

const inputText = ref('')

function handleSend() {
  if (!inputText.value.trim()) return
  emit('send', inputText.value)
  inputText.value = ''
}
</script>

<style scoped lang="scss">
.chat-input-area {
  padding: 12px;
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.chat-input {
  :deep(.q-field__control) {
    background: var(--color-bg-primary);
    border-color: var(--color-border);
    border-radius: 24px;
    padding-right: 4px;
  }
}
</style>
