import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/auth'

export const apiClient = axios.create({
  baseURL: '/api',
  withCredentials: false,
})

apiClient.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }
  return config
})

let refreshPromise: Promise<string | null> | null = null

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retry?: boolean }) | undefined
    if (error.response?.status !== 401 || !original || original._retry) {
      throw error
    }

    const auth = useAuthStore()
    if (!auth.refreshToken) {
      auth.clearSession()
      throw error
    }

    original._retry = true
    if (!refreshPromise) {
      refreshPromise = auth
        .refreshAccessToken()
        .then((token) => token)
        .catch(() => {
          auth.clearSession()
          return null
        })
        .finally(() => {
          refreshPromise = null
        })
    }

    const newToken = await refreshPromise
    if (!newToken) {
      throw error
    }

    original.headers.Authorization = `Bearer ${newToken}`
    return apiClient(original)
  },
)
