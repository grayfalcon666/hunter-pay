import apiClient from 'src/api/client'

// 创建邀请（悬赏者 -> 猎人）
export async function createInvitation(bountyId, hunterUsername) {
  const data = await apiClient.post(`/bounties/${bountyId}/invitations`, {
    bounty_id: bountyId,
    hunter_username: hunterUsername,
  })
  return data.invitation || data
}

// 获取我收到的邀请（猎人视角）
export async function getMyInvitations({ status = '', pageId = 1, pageSize = 20 } = {}) {
  const params = { page_id: pageId, page_size: pageSize }
  if (status) params.status = status
  const data = await apiClient.get('/invitations/received', { params })
  return data
}

// 获取我发出的邀请（悬赏者视角）
export async function getMySentInvitations({ bountyId = 0, status = '', pageSize = 20 } = {}) {
  const params = { page_size: pageSize }
  if (bountyId > 0) params.bounty_id = bountyId
  if (status) params.status = status
  const data = await apiClient.get('/invitations/sent', { params })
  return data
}

// 猎人响应邀请（接受/拒绝）
export async function respondToInvitation(invitationId, accept) {
  const data = await apiClient.post(`/invitations/${invitationId}/respond`, {
    invitation_id: invitationId,
    accept,
  })
  return data
}

// 获取我发出的申请（猎人视角）
export async function getMyApplications({ status = '', pageId = 1, pageSize = 20 } = {}) {
  const params = { page_id: pageId, page_size: pageSize }
  if (status) params.status = status
  const data = await apiClient.get('/bounties/applications/received', { params })
  return data
}
