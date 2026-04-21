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
  const commentsFlat = ref([]) // 完整扁平列表（含根评论和回复），用于 replyToId 查找
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
      console.log('[bountyStore] fetching bounty:', id)
      const data = await apiClient.get(`/bounties/${id}`)
      console.log('[bountyStore] raw response:', JSON.stringify(data))
      const bounty = data.bounty || data
      // Attach applications to bounty object (API returns {bounty, applications} separately)
      if (data.applications) {
        bounty.applications = data.applications
      }
      console.log('[bountyStore] parsed bounty:', JSON.stringify(bounty))
      currentBounty.value = bounty
    } catch (e) {
      console.error('[bountyStore] fetchBounty error:', e)
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
      console.log('[bountyStore] fetching comments for bounty:', bountyId)
      const data = await apiClient.get(`/bounties/${bountyId}/comments`)
      console.log('[bountyStore] comments raw response:', JSON.stringify(data))
      // 后端返回扁平列表：根评论（parentId=null）包含 replyToId
      // 前端按 parentId 手动构建 replies 树
      const allComments = data.comments || []
      // 构建 id -> 评论的映射，用于查找 reply_to 作者
      const idMap = {}
      allComments.forEach(c => { idMap[Number(c.id)] = c })
      // 给每个评论补上 replyToUsername（显示"回复 @xxx"）
      allComments.forEach(c => {
        const replyToId = c.replyToId != null ? Number(c.replyToId) : null
        if (replyToId && idMap[replyToId]) {
          c.replyToUsername = idMap[replyToId].authorUsername
        }
      })
      // 构建 replies 树：parentId == null 或 "0" 才是父亲评论
      allComments.forEach(c => { c.replies = [] })
      allComments.forEach(c => {
        const parentId = c.parentId != null ? Number(c.parentId) : null
        // parentId 为 0 或 null 表示根评论，跳过
        if (parentId != null && parentId !== 0 && idMap[parentId]) {
          idMap[parentId].replies.push(c)
        }
      })
      // 根评论：parentId 为 null 或 "0" 或 0
      comments.value = allComments.filter(c => c.parentId == null || c.parentId === '0' || c.parentId === 0)
      // 保存完整扁平列表
      commentsFlat.value = allComments
      console.log('[bountyStore] fetchComments final root comments:', JSON.stringify(comments.value))
      console.log('[bountyStore] fetchComments all comments with replies:', JSON.stringify(allComments.map(c => ({ id: c.id, parentId: c.parentId, replyToId: c.replyToId, content: c.content, replies: c.replies?.length }))))
    } catch (e) {
      console.error('[bountyStore] fetchComments error:', e)
    } finally {
      commentsLoading.value = false
    }
  }

  async function addComment(bountyId, { parent_id, reply_to_id, content, imageId }) {
    const body = {
      parent_id: parent_id ?? 0,
      reply_to_id: reply_to_id ?? 0,
      content,
      image_id: imageId ?? 0,
    }
    console.log('[bountyStore] addComment called, bountyId:', bountyId)
    console.log('[bountyStore] addComment received params:', { parent_id, reply_to_id, content, imageId })
    console.log('[bountyStore] addComment body to send:', JSON.stringify(body))
    const data = await apiClient.post(`/bounties/${bountyId}/comments`, body)
    console.log('[bountyStore] addComment response:', JSON.stringify(data))
    const newComment = data.comment
    // 统一转为 number 避免类型比较问题
    const parentIdNum = newComment.parentId != null ? Number(newComment.parentId) : null
    const replyToIdNum = newComment.replyToId != null ? Number(newComment.replyToId) : null
    // 补上 replyToUsername（reply_to_id 决定下方引用谁）
    if (replyToIdNum && comments.value) {
      const allComments = []
      comments.value.forEach(c => {
        allComments.push(c)
        if (c.replies) c.replies.forEach(r => allComments.push(r))
      })
      const idMap = {}
      allComments.forEach(c => { idMap[Number(c.id)] = c })
      if (idMap[replyToIdNum]) {
        newComment.replyToUsername = idMap[replyToIdNum].authorUsername
      }
    }
    // parent_id == null 或 0 才是父亲评论，挂在根评论列表
    if (parentIdNum == null || parentIdNum === 0) {
      comments.value = [...comments.value, newComment]
    } else {
      // 挂到对应的根评论下
      const root = comments.value.find(c => Number(c.id) === parentIdNum)
      if (root) {
        root.replies = [...(root.replies || []), newComment]
        console.log('[bountyStore] addComment appended to root replies, root id:', root.id, 'replies count:', root.replies.length)
      } else {
        console.log('[bountyStore] addComment ERROR: root not found for parentId:', parentIdNum)
      }
    }
    // 新评论加入扁平列表
    commentsFlat.value.push(newComment)
    console.log('[bountyStore] addComment final root comments:', JSON.stringify(comments.value.map(c => ({ id: c.id, parentId: c.parentId, replies: c.replies?.length }))))
    return newComment
  }

  async function removeComment(commentId) {
    await apiClient.delete(`/comments/${commentId}`)
    // 从根评论列表及其 replies 中删除
    const cid = Number(commentId)
    comments.value = comments.value
      .filter(c => Number(c.id) !== cid)
      .map(c => ({
        ...c,
        replies: (c.replies || []).filter(r => Number(r.id) !== cid),
      }))
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
    commentsFlat,
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
