<template>
  <Teleport to="body">
    <div v-if="chatStore.chatOpen" class="chat-overlay" @click.self="chatStore.toggleMinimize">
      <div class="chat-drawer" :class="{ minimized: chatStore.chatMinimized }">

        <!-- Header -->
        <div class="chat-header">
          <q-btn
            v-if="chatStore.activePanel === 'chat'"
            flat dense round icon="arrow_back"
            class="chat-back-btn"
            @click="chatStore.leaveConversation"
          />
          <div class="chat-header-title">
            <q-icon name="chat_bubble" size="18px" />
            <span>{{ chatStore.activePanel === 'chat' ? otherUsername : '私信' }}</span>
          </div>
          <q-btn flat dense round icon="minimize" @click="chatStore.toggleMinimize" />
          <q-btn flat dense round icon="close" @click="chatStore.closeChat" />
        </div>

        <!-- Conversation List Panel -->
        <div v-if="chatStore.activePanel === 'list'" class="chat-list-panel">

          <!-- Search -->
          <div class="chat-search">
            <q-input
              v-model="chatStore.searchQuery"
              dense outlined
              placeholder="搜索用户..."
              class="search-input"
              @update:model-value="handleSearch"
            >
              <template #prepend>
                <q-icon name="search" size="sm" />
              </template>
            </q-input>

            <!-- Search Results -->
            <div v-if="chatStore.searchResults.length" class="search-results">
              <div
                v-for="user in chatStore.searchResults"
                :key="user.username"
                class="search-result-item"
                @click="startChat(user.username)"
              >
                <div class="user-avatar">{{ user.username[0].toUpperCase() }}</div>
                <span>{{ user.username }}</span>
              </div>
            </div>
          </div>

          <!-- Conversation List -->
          <div class="conversations-list">
            <div
              v-if="chatStore.conversations.length === 0"
              class="empty-state"
            >
              <q-icon name="chat_bubble_outline" size="40px" />
              <p>暂无会话</p>
              <p class="hint">搜索用户开始聊天</p>
            </div>
            <ConversationItem
              v-for="conv in chatStore.conversations"
              :key="conv.id"
              :conversation="conv"
              :active="conv.id === chatStore.activeConvId"
              @click="chatStore.openConversation(conv.id)"
              @delete="chatStore.deleteConversation(conv.id)"
            />
          </div>
        </div>

        <!-- Chat Panel -->
        <div v-if="chatStore.activePanel === 'chat'" class="chat-messages-panel">
          <div class="messages-list" ref="messagesListRef">
            <ChatBubble
              v-for="msg in currentMessages"
              :key="msg.id"
              :message="msg"
              :is-own="(msg.sender_username || msg.senderUsername) === authStore.username"
              class="debug-bubble"
            />
            <div v-if="currentMessages.length === 0" class="empty-chat">
              <p>暂无消息，开始聊天吧</p>
            </div>
          </div>
          <ChatInput @send="handleSend" />
        </div>

      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useChatStore } from 'src/stores/chat'
import { useAuthStore } from 'src/stores/auth'
import ConversationItem from './ConversationItem.vue'
import ChatBubble from './ChatBubble.vue'
import ChatInput from './ChatInput.vue'

const chatStore = useChatStore()
const authStore = useAuthStore()
const messagesListRef = ref(null)

const currentMessages = computed(() => {
  if (!chatStore.activeConvId) return []
  const msgs = chatStore.messagesByConv[chatStore.activeConvId] || []
  return Array.isArray(msgs) ? [...msgs] : []
})

const otherUsername = computed(() => {
  if (!chatStore.activeConvId) return ''
  const conv = chatStore.conversations.find(c => c.id === chatStore.activeConvId)
  return conv?.otherUsername || ''
})

// Auto-scroll to bottom on new messages
watch(() => currentMessages.value.length, async () => {
  await nextTick()
  if (messagesListRef.value) {
    messagesListRef.value.scrollTop = messagesListRef.value.scrollHeight
  }
})

function handleSearch(query) {
  chatStore.searchUsers(query)
}

async function startChat(username) {
  await chatStore.startConversation(username)
}

function handleSend(content) {
  console.log('handleSend called, content:', content, 'activeConvId:', chatStore.activeConvId)
  if (chatStore.activeConvId) {
    chatStore.sendMessage(chatStore.activeConvId, content)
  }
}
</script>

<style scoped lang="scss">
.chat-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  pointer-events: none;
}

.chat-drawer {
  position: fixed;
  bottom: 0;
  right: 24px;
  width: 380px;
  max-height: 580px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 16px 16px 0 0;
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  pointer-events: all;
  overflow: hidden;
  transition: transform 0.2s, max-height 0.2s;

  &.minimized {
    max-height: 52px;
    overflow: hidden;
  }

  @media (max-width: 480px) {
    right: 0;
    width: 100vw;
    max-height: 85vh;
    border-radius: 16px 16px 0 0;
    padding-bottom: env(safe-area-inset-bottom);
  }
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--color-bg-elevated);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.chat-back-btn {
  color: var(--color-text-muted);
}

.chat-header-title {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--color-text-primary);
}

// Search
.chat-search {
  position: relative;
  padding: 12px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-border);
}

.search-input {
  :deep(.q-field__control) {
    background: var(--color-bg-primary);
    border-color: var(--color-border);
  }
}

.search-results {
  position: absolute;
  top: calc(100% - 4px);
  left: 12px;
  right: 12px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  z-index: 10;
  max-height: 200px;
  overflow-y: auto;
}

.search-result-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: var(--color-bg-primary);
  }
}

.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  flex-shrink: 0;
}

// Conversation List
.conversations-list {
  flex: 1;
  overflow-y: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--color-text-muted);
  gap: 8px;

  p { margin: 0; }
  .hint { font-size: 0.8rem; }
}

// Chat Messages
.chat-messages-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.messages-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.empty-chat {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 0.85rem;
  p { margin: 0; }
}
</style>
