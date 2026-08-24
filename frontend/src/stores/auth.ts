import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import * as authApi from '@/api/auth'

const ACCESS_TOKEN_KEY = 'lmqr_access_token'
const REFRESH_TOKEN_KEY = 'lmqr_refresh_token'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(localStorage.getItem(ACCESS_TOKEN_KEY))
  const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY))
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'ADMIN')
  const isClient = computed(() => user.value?.role === 'CLIENT')

  function setSession(token: string, refresh: string, sessionUser: User) {
    accessToken.value = token
    refreshToken.value = refresh
    user.value = sessionUser
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
    localStorage.setItem(REFRESH_TOKEN_KEY, refresh)
  }

  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }

  async function login(email: string, password: string) {
    const res = await authApi.login(email, password)
    setSession(res.access_token, res.refresh_token, res.user)
  }

  async function logout() {
    if (refreshToken.value) {
      try {
        await authApi.logout(refreshToken.value)
      } catch {
        // best-effort revoke
      }
    }
    clearSession()
  }

  async function refreshAccessToken(): Promise<string | null> {
    if (!refreshToken.value) return null
    const res = await authApi.refresh(refreshToken.value)
    setSession(res.access_token, res.refresh_token, res.user)
    return res.access_token
  }

  async function fetchCurrentUser() {
    user.value = await authApi.me()
  }

  return {
    accessToken,
    refreshToken,
    user,
    isAuthenticated,
    isAdmin,
    isClient,
    setSession,
    clearSession,
    login,
    logout,
    refreshAccessToken,
    fetchCurrentUser,
  }
})
