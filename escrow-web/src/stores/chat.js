import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as chatApi from 'src/api/chat'
import { useAuthStore } from 'src/stores/auth'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'
const RECONNECT_BASE_MS = 1000
const MAX_RECONNECT_MS = 30000
const HEARTBEAT_INTERVAL = 25000

export const useChatStore = defineStore('chat', () => {
  const authStore = useAuthStore()

  // State
  const conversations = ref([])
  const activeConvId = ref(null)
  const messagesByConv = ref({})
  const unreadByConv = ref({})
  const totalUnread = computed(() =>
    Object.values(unreadByConv.value).reduce((a, b) => a + b, 0)
  )

  // WebSocket state
  let ws = null
  let reconnectAttempts = 0
  let reconnectTimer = null
  let heartbeatTimer = null

  // UI state
  const chatOpen = ref(false)
  const chatMinimized = ref(false)
  const activePanel = ref('list') // 'list' | 'chat'
  const searchQuery = ref('')
  const searchResults = ref([])
  const searchLoading = ref(false)

  // ================== WebSocket ==================

  function connectWS() {
    if (!authStore.isLoggedIn()) return
    const token = localStorage.getItem('token')
    if (!token) return

    console.log('Connecting to WebSocket:', WS_URL)
    try {
      ws = new WebSocket(`${WS_URL}?token=Bearer ${token}`)
    } catch (e) {
      console.error('WebSocket creation failed:', e)
      scheduleReconnect()
      return
    }

    ws.onopen = () => {
      console.log('[DEBUG] WebSocket connected!')
      console.log('[DEBUG] conversations to rejoin:', conversations.value.length)
      reconnectAttempts = 0
      startHeartbeat()
      // Rejoin active conversations
      conversations.value.forEach(c => {
        console.log('[DEBUG] joining conversation:', c.id)
        sendWS({ action: 'join_conv', payload: { conversation_id: c.id } })
      })
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        handleWSMessage(msg)
      } catch {
        console.error('WS parse error')
      }
    }

    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason)
      stopHeartbeat()
      scheduleReconnect()
    }
    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  function scheduleReconnect() {
    clearTimeout(reconnectTimer)
    const delay = Math.min(RECONNECT_BASE_MS * 2 ** reconnectAttempts, MAX_RECONNECT_MS)
    reconnectAttempts++
    reconnectTimer = setTimeout(connectWS, delay)
  }

  function disconnectWS() {
    clearTimeout(reconnectTimer)
    stopHeartbeat()
    if (ws) {
      ws.onclose = null
      ws.close()
      ws = null
    }
  }

  function startHeartbeat() {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ action: 'ping' }))
      }
    }, HEARTBEAT_INTERVAL)
  }

  function stopHeartbeat() {
    clearInterval(heartbeatTimer)
  }

  function sendWS(payload) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(payload))
    } else {
      console.error('WebSocket not connected, state:', ws?.readyState)
    }
  }

  function handleWSMessage(msg) {
    switch (msg.type) {
      case 'new_message': {
        console.log('[DEBUG] new_message received:', msg.payload)
        appendMessage(msg.payload)
        if (msg.payload.conversation_id !== activeConvId.value) {
          const newUnreads = { ...unreadByConv.value }
          newUnreads[msg.payload.conversation_id] =
            (newUnreads[msg.payload.conversation_id] || 0) + 1
          unreadByConv.value = newUnreads
        }
        break
      }
      case 'message_sent': {
        console.log('[DEBUG] message_sent received:', msg.payload)
        const sentMsg = msg.payload?.message
        const tempId = msg.payload?.temp_id || ''
        if (sentMsg) {
          // temp_id 为空时，尝试用 message.id 替换最新的乐观消息
          replaceMessageByContent(sentMsg, tempId)
        }
        break
      }
      case 'unread_update': {
        const newUnreads2 = { ...unreadByConv.value }
        newUnreads2[msg.payload.conversation_id] = msg.payload.unread_count
        unreadByConv.value = newUnreads2
        break
      }
    }
  }

  function replaceMessageByContent(serverMsg, tempId) {
    const cid = String(serverMsg.conversation_id)
    const msgs = messagesByConv.value[cid]
    if (!msgs) return

    // 如果有 temp_id，用它精确匹配；否则找最后一个相同内容的乐观消息
    let idx = -1
    if (tempId) {
      idx = msgs.findIndex(m => m.id === tempId)
    } else {
      // 找到最后一条发送者相同、内容相同的临时消息
      for (let i = msgs.length - 1; i >= 0; i--) {
        const m = msgs[i]
        if (m.sender_username === serverMsg.sender_username &&
            m.content === serverMsg.content &&
            String(m.id).startsWith('temp-')) {
          idx = i
          break
        }
      }
    }

    if (idx !== -1) {
      const newMsgs = [...msgs]
      newMsgs[idx] = {
        ...serverMsg,
        sender_username: serverMsg.sender_username || serverMsg.senderUsername,
      }
      const newAll = { ...messagesByConv.value }
      newAll[cid] = newMsgs
      messagesByConv.value = newAll
    }
  }

  function appendMessage(msg) {
    console.log('[DEBUG] appendMessage called:', msg)
    const normalizedMsg = {
      ...msg,
      sender_username: msg.sender_username || msg.senderUsername,
    }
    const convId = String(normalizedMsg.conversation_id)
    console.log('[DEBUG] convId:', convId, 'existing msgs:', messagesByConv.value[convId]?.length)

    const newMessages = { ...messagesByConv.value }
    if (!newMessages[convId]) {
      newMessages[convId] = []
    }
    if (!newMessages[convId].find(m => m.id === normalizedMsg.id)) {
      newMessages[convId] = [...newMessages[convId], normalizedMsg]
    }
    messagesByConv.value = newMessages
  }

  // ================== Conversations ==================

  async function loadConversations() {
    try {
      const data = await chatApi.listConversations()
      conversations.value = data.conversations || []
      // 构建新的 unread 对象并替换
      const newUnreads = {}
      conversations.value.forEach(c => {
        newUnreads[c.id] = c.unread_count || 0
      })
      unreadByConv.value = newUnreads
    } catch (e) {
      console.error('loadConversations error:', e)
    }
  }

  async function startConversation(otherUsername) {
    const data = await chatApi.getOrCreateConversation(otherUsername)
    const conv = data.conversation
    await loadConversations()
    await openConversation(conv.id)
    searchResults.value = []
    searchQuery.value = ''
  }

  async function openConversation(convId) {
    activeConvId.value = convId
    chatOpen.value = true
    chatMinimized.value = false
    activePanel.value = 'chat'

    // 加载消息时也需要触发响应式更新
    const newMessages = { ...messagesByConv.value }
    if (!newMessages[convId]) {
      newMessages[convId] = []
    }
    if (newMessages[convId].length === 0) {
      try {
        const data = await chatApi.listPrivateMessages(convId)
        newMessages[convId] = (data.messages || []).map(msg => ({
          ...msg,
          sender_username: msg.sender_username || msg.senderUsername
        }))
      } catch (e) {
        console.error('load messages error:', e)
      }
    }
    messagesByConv.value = newMessages

    try {
      await chatApi.markRead(convId)
    } catch {
      // ignore
    }
    // 触发 unreadByConv 响应式更新
    const newUnreads = { ...unreadByConv.value }
    newUnreads[convId] = 0
    unreadByConv.value = newUnreads
    sendWS({ action: 'join_conv', payload: { conversation_id: convId } })
  }

  function leaveConversation() {
    if (activeConvId.value !== null) {
      sendWS({ action: 'leave_conv', payload: { conversation_id: activeConvId.value } })
    }
    activeConvId.value = null
    activePanel.value = 'list'
  }

  async function sendMessage(convId, content) {
    if (!content.trim()) return
    const tempId = `temp-${Date.now()}`
    const optimisticMsg = {
      id: tempId,
      conversation_id: convId,
      sender_username: authStore.username,
      content,
      is_read: false,
      created_at: Math.floor(Date.now() / 1000),
    }
    appendMessage(optimisticMsg)
    const wsPayload = { action: 'send_message', payload: { conversation_id: convId, content } }
    console.log('sendMessage, wsPayload:', JSON.stringify(wsPayload), 'ws.readyState:', ws?.readyState)
    sendWS(wsPayload)
  }

  async function deleteConversation(convId) {
    try {
      await chatApi.deleteConversation(convId)
    } catch (e) {
      console.error('delete conversation error:', e)
    }
    conversations.value = conversations.value.filter(c => c.id !== convId)
    delete messagesByConv.value[convId]
    delete unreadByConv.value[convId]
    if (activeConvId.value === convId) {
      leaveConversation()
    }
  }

  // ================== User Search ==================

  async function searchUsers(query) {
    if (!query || query.length < 2) {
      searchResults.value = []
      return
    }
    searchLoading.value = true
    try {
      const res = await fetch(`/api/v1/users/search?q=${encodeURIComponent(query)}`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
      })
      const data = await res.json()
      searchResults.value = data.users || []
    } catch {
      searchResults.value = []
    } finally {
      searchLoading.value = false
    }
  }

  // ================== UI Controls ==================

  function openChat() {
    chatOpen.value = true
    chatMinimized.value = false
  }

  function toggleMinimize() {
    chatMinimized.value = !chatMinimized.value
  }

  function closeChat() {
    leaveConversation()
    chatOpen.value = false
    chatMinimized.value = false
  }

  // ================== Init ==================

  async function init() {
    if (authStore.isLoggedIn()) {
      await loadConversations()
      connectWS()
    }
  }

  function cleanup() {
    disconnectWS()
    conversations.value = []
    messagesByConv.value = {}
    unreadByConv.value = {}
    activeConvId.value = null
    chatOpen.value = false
  }

  return {
    conversations,
    activeConvId,
    messagesByConv,
    unreadByConv,
    totalUnread,
    chatOpen,
    chatMinimized,
    activePanel,
    searchQuery,
    searchResults,
    searchLoading,
    connectWS,
    disconnectWS,
    loadConversations,
    startConversation,
    openConversation,
    leaveConversation,
    sendMessage,
    deleteConversation,
    searchUsers,
    openChat,
    toggleMinimize,
    closeChat,
    init,
    cleanup,
  }
})
