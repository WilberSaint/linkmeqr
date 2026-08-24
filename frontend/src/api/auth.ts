import { apiClient } from './client'
import type { MeResponse, User } from '@/types'

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: User
}

export function login(email: string, password: string) {
  return apiClient.post<LoginResponse>('/auth/login', { email, password }).then((r) => r.data)
}

export function refresh(refreshToken: string) {
  return apiClient
    .post<LoginResponse>('/auth/refresh', { refresh_token: refreshToken })
    .then((r) => r.data)
}

export function logout(refreshToken: string) {
  return apiClient.post('/auth/logout', { refresh_token: refreshToken })
}

export function me() {
  return apiClient.get<MeResponse>('/me').then((r) => r.data)
}
