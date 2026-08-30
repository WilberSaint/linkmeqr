import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import * as authApi from '@/api/auth'

const ACCESS_TOKEN_KEY = 'lmqr_access_token'
const REFRESH_TOKEN_KEY = 'lmqr_refresh_token'
const IMPERSONATION_KEY = 'lmqr_admin_session'

interface StashedAdminSession {
  accessToken: string
  refreshToken: string
  user: User
}

function loadStashedAdminSession(): StashedAdminSession | null {
  const raw = localStorage.getItem(IMPERSONATION_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as StashedAdminSession
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(localStorage.getItem(ACCESS_TOKEN_KEY))
  const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY))
  const user = ref<User | null>(null)
  const impersonation = ref<StashedAdminSession | null>(loadStashedAdminSession())

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'ADMIN')
  const isClient = computed(() => user.value?.role === 'CLIENT')
  const isImpersonating = computed(() => !!impersonation.value)

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
    impersonation.value = null
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(IMPERSONATION_KEY)
  }

  // Stashes the current (admin) session, then switches to a CLIENT-scoped
  // token obtained via POST /admin/clients/:id/impersonate. No refresh token
  // is issued for the impersonated session — it simply expires.
  function startImpersonation(clientAccessToken: string, clientUser: User) {
    if (!accessToken.value || !refreshToken.value || !user.value) return
    const stashed: StashedAdminSession = {
      accessToken: accessToken.value,
      refreshToken: refreshToken.value,
      user: user.value,
    }
    impersonation.value = stashed
    localStorage.setItem(IMPERSONATION_KEY, JSON.stringify(stashed))

    accessToken.value = clientAccessToken
    refreshToken.value = null
    user.value = clientUser
    localStorage.setItem(ACCESS_TOKEN_KEY, clientAccessToken)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }

  function stopImpersonation() {
    const stashed = impersonation.value
    if (!stashed) return
    setSession(stashed.accessToken, stashed.refreshToken, stashed.user)
    impersonation.value = null
    localStorage.removeItem(IMPERSONATION_KEY)
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
    impersonation,
    isAuthenticated,
    isAdmin,
    isClient,
    isImpersonating,
    setSession,
    clearSession,
    login,
    logout,
    refreshAccessToken,
    fetchCurrentUser,
    startImpersonation,
    stopImpersonation,
  }
})
