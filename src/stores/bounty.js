import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from 'src/api/client'
import { useAuthStore } from 'src/stores/auth'

export const useBountyStore = defineStore('bounty', () => {
  const bounties = ref([])
  const currentBounty = ref(null)
  const total = ref(0)
  const loading = ref(false)
  const error = ref(null)

  // Comments
  const comments = ref([])
  const commentsLoading = ref(false)

  // 悬赏列表
  async function fetchBounties({ status = '', page = 1, pageSize = 10 } = {}) {
    loading.value = true
    error.value = null
    try {
      const params = { page_size: pageSize }
      if (page > 1) params.page_id = page
      if (status) params.status = status

      const data = await apiClient.get('/bounties', { params })
      bounties.value = data.bounties || data
      total.value = data.total || bounties.value.length
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  // 悬赏详情
  async function fetchBounty(id) {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get(`/bounties/${id}`)
      const bounty = data.bounty || data
      // Attach applications to bounty object (API returns {bounty, applications} separately)
      if (data.applications) {
        bounty.applications = data.applications
      }
      currentBounty.value = bounty
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  // 创建悬赏
  async function createBounty(payload) {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.post('/bounties', payload)
      return data.bounty || data
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  // 接单（后端通过 JWT Token 自动获取猎人账户 ID）
  async function acceptBounty(bountyId) {
    const data = await apiClient.post(`/bounties/${bountyId}/accept`, {})
    return data.application || data
  }

  // 确认猎人
  async function confirmHunter(bountyId, applicationId) {
    const data = await apiClient.post(`/bounties/${bountyId}/confirm`, {
      application_id: applicationId,
    })
    return data.bounty || data
  }

  // 完成悬赏
  async function completeBounty(bountyId) {
    const data = await apiClient.post(`/bounties/${bountyId}/complete`)
    return data.bounty || data
  }

  // 猎人提交工作成果
  async function submitBounty(bountyId, submissionText) {
    const data = await apiClient.post(`/bounties/${bountyId}/submit`, {
      submission_text: submissionText,
    })
    return data.bounty || data
  }

  // 雇主审核通过
  async function approveBounty(bountyId) {
    const data = await apiClient.post(`/bounties/${bountyId}/approve`)
    return data.bounty || data
  }

  // 雇主审核拒绝
  async function rejectBounty(bountyId) {
    const data = await apiClient.post(`/bounties/${bountyId}/reject`)
    return data.bounty || data
  }

  // 取消悬赏
  async function cancelBounty(bountyId) {
    const data = await apiClient.post(`/bounties/${bountyId}/cancel`)
    return data.bounty || data
  }

  // 删除悬赏（仅 PENDING/FAILED）
  async function deleteBounty(bountyId) {
    const data = await apiClient.delete(`/bounties/${bountyId}`)
    return data
  }

  // Comments
  async function fetchComments(bountyId) {
    commentsLoading.value = true
    try {
      const data = await apiClient.get(`/bounties/${bountyId}/comments`)
      comments.value = data.comments || []
    } catch (e) {
      console.error('fetchComments error:', e)
    } finally {
      commentsLoading.value = false
    }
  }

  async function addComment(bountyId, { parentId, content }) {
    const data = await apiClient.post(`/bounties/${bountyId}/comments`, {
      parent_id: parentId || 0,
      content,
    })
    if (parentId) {
      // Find parent comment and push to its replies
      const parent = comments.value.find(c => c.id === parentId)
      if (parent) {
        if (!parent.replies) parent.replies = []
        parent.replies = [...parent.replies, data.comment]
      }
    } else {
      comments.value = [...comments.value, data.comment]
    }
    return data.comment
  }

  async function removeComment(commentId) {
    await apiClient.delete(`/comments/${commentId}`)
    // Recursively remove comment and its replies from nested structure
    function removeFromTree(list) {
      return list.filter(c => {
        if (c.id === commentId) return false
        if (c.parent_id === commentId) return false
        if (c.replies) c.replies = removeFromTree(c.replies)
        return true
      })
    }
    comments.value = removeFromTree(comments.value)
  }

  // 提交互评
  async function submitReview({ reviewedUsername, bountyId, rating, comment }) {
    const authStore = useAuthStore()
    const role = authStore.isPoster ? 'EMPLOYER_TO_HUNTER' : 'HUNTER_TO_EMPLOYER'
    const data = await apiClient.post('/reviews', {
      reviewed_username: reviewedUsername,
      bounty_id: bountyId,
      rating,
      comment: comment || '',
      review_type: role,
    })
    return data.review || data
  }

  function resetState() {
    bounties.value = []
    currentBounty.value = null
    total.value = 0
    loading.value = false
    error.value = null
  }

  return {
    bounties,
    currentBounty,
    total,
    loading,
    error,
    comments,
    commentsLoading,
    fetchBounties,
    fetchBounty,
    createBounty,
    acceptBounty,
    confirmHunter,
    completeBounty,
    submitBounty,
    approveBounty,
    rejectBounty,
    cancelBounty,
    deleteBounty,
    fetchComments,
    addComment,
    removeComment,
    submitReview,
    resetState,
  }
})
