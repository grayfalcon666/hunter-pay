import apiClient from 'src/api/client'

// ================== Private Chat ==================

export async function listConversations() {
  return apiClient.get('/conversations')
}

export async function getOrCreateConversation(otherUsername) {
  return apiClient.post('/conversations', { other_username: otherUsername })
}

export async function deleteConversation(convId) {
  return apiClient.delete(`/conversations/${convId}`)
}

export async function listPrivateMessages(convId, beforeId = 0, limit = 50) {
  return apiClient.get(`/conversations/${convId}/messages`, {
    params: { conversation_id: convId, before_id: beforeId, limit }
  })
}

export async function markRead(convId) {
  return apiClient.post(`/conversations/${convId}/read`)
}

export async function getUnreadCounts() {
  return apiClient.get('/conversations/unread')
}

// ================== Threaded Comments ==================

export async function listComments(bountyId) {
  return apiClient.get(`/bounties/${bountyId}/comments`)
}

export async function createComment(bountyId, { replyToId, content, imageId }) {
  const body = {
    reply_to_id: replyToId || 0,
    content,
  }
  if (imageId) body.image_id = imageId
  console.log('[chat.createComment] POST /bounties/' + bountyId + '/comments, body:', JSON.stringify(body))
  return apiClient.post(`/bounties/${bountyId}/comments`, body)
}

export async function deleteComment(commentId) {
  return apiClient.delete(`/comments/${commentId}`)
}

export async function listUserComments(username, page = 1, pageSize = 20) {
  return apiClient.get('/comments', {
    params: { username, page_id: page, page_size: pageSize },
  })
}
