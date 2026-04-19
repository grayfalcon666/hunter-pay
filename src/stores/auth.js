import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const role = ref(localStorage.getItem('role') || 'EMPLOYER') // EMPLOYER | HUNTER

  const isLoggedIn = () => !!token.value
  const isHunter = computed(() => role.value === 'HUNTER')
  const isPoster = computed(() => role.value === 'EMPLOYER')

  function setAuth(accessToken, user) {
    token.value = accessToken
    username.value = user
    localStorage.setItem('token', accessToken)
    localStorage.setItem('username', user)
  }

  function setRole(newRole) {
    role.value = newRole
    localStorage.setItem('role', newRole)
  }

  function switchRole() {
    const newRole = role.value === 'EMPLOYER' ? 'HUNTER' : 'EMPLOYER'
    setRole(newRole)
  }

  function logout() {
    token.value = ''
    username.value = ''
    role.value = 'EMPLOYER'
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
  }

  return { token, username, role, isLoggedIn, isHunter, isPoster, setAuth, setRole, switchRole, logout }
})
