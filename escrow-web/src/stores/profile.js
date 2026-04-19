import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from 'src/api/client'

export const useProfileStore = defineStore('profile', () => {
  const profile = ref(null)
  const reviews = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchProfile(username) {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get(`/profiles/${username}`)
      profile.value = data.profile || data
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function createProfile(payload) {
    const data = await apiClient.post('/profiles', payload)
    profile.value = data.profile || data
    return profile.value
  }

  async function updateProfile(username, payload) {
    const data = await apiClient.put(`/profiles/${username}`, payload)
    profile.value = data.profile || data
    return profile.value
  }

  async function fetchReviews(username) {
    loading.value = true
    try {
      const data = await apiClient.get(`/users/${username}/reviews`)
      reviews.value = data.reviews || data || []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function createReview(payload) {
    const data = await apiClient.post('/reviews', payload)
    return data.review || data
  }

  return { profile, reviews, loading, error, fetchProfile, createProfile, updateProfile, fetchReviews, createReview }
})
