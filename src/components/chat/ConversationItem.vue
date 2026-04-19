<template>
  <div class="conv-item" :class="{ active }" @click="$emit('click')">
    <div class="conv-avatar">
      {{ conversation.otherUsername?.[0]?.toUpperCase() || '?' }}
    </div>
    <div class="conv-info">
      <div class="conv-name">{{ conversation.otherUsername }}</div>
      <div class="conv-preview">{{ conversation.lastMessageContent || '暂无消息' }}</div>
    </div>
    <div class="conv-meta">
      <q-badge
        v-if="conversation.unreadCount > 0"
        color="red"
        :label="conversation.unreadCount > 99 ? '99+' : conversation.unreadCount"
        class="unread-badge"
      />
      <q-btn
        flat dense round icon="more_vert"
        size="sm"
        @click.stop="showMenu = true"
      />
      <q-menu v-model="showMenu" context-menu>
        <q-list dense style="min-width: 120px">
          <q-item clickable v-close-popup @click="$emit('delete')">
            <q-item-section avatar><q-icon name="delete" size="sm" color="negative" /></q-item-section>
            <q-item-section>删除会话</q-item-section>
          </q-item>
        </q-list>
      </q-menu>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  conversation: { type: Object, required: true },
  active: { type: Boolean, default: false },
})
defineEmits(['click', 'delete'])

const showMenu = ref(false)
</script>

<style scoped lang="scss">
.conv-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.15s;
  border-bottom: 1px solid var(--color-border);

  &:hover { background: var(--color-bg-primary); }
  &.active { background: var(--color-bg-elevated); }
}

.conv-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-accent-gold);
  flex-shrink: 0;
}

.conv-info {
  flex: 1;
  min-width: 0;
}

.conv-name {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-preview {
  font-size: 0.8rem;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.unread-badge {
  font-size: 0.7rem;
}
</style>
