import { apiClient } from './client'
import type { ClientWithLicense, DurationType, License, Profile, User } from '@/types'

export function listClients() {
  return apiClient.get<ClientWithLicense[]>('/admin/clients/').then((r) => r.data)
}

export function createClient(payload: { email: string; password: string; full_name: string; phone?: string | null }) {
  return apiClient.post<User>('/admin/clients/', payload).then((r) => r.data)
}

export function getClient(id: string) {
  return apiClient.get<User>(`/admin/clients/${id}`).then((r) => r.data)
}

export function updateClient(id: string, payload: { full_name: string; phone?: string | null }) {
  return apiClient.patch<User>(`/admin/clients/${id}`, payload).then((r) => r.data)
}

export function activateClient(id: string) {
  return apiClient.post(`/admin/clients/${id}/activate`)
}

export function deactivateClient(id: string) {
  return apiClient.post(`/admin/clients/${id}/deactivate`)
}

export function activateLicenseForClient(
  clientId: string,
  payload: { duration_type: DurationType; custom_days?: number },
) {
  return apiClient
    .post<License>(`/admin/clients/${clientId}/license/activate`, payload)
    .then((r) => r.data)
}

export function getClientProfile(clientId: string) {
  return apiClient.get<Profile>(`/admin/clients/${clientId}/profile`).then((r) => r.data)
}

export function createClientProfile(clientId: string, payload: { business_name: string; slug?: string }) {
  return apiClient.post<Profile>(`/admin/clients/${clientId}/profile`, payload).then((r) => r.data)
}

export interface ImpersonateResponse {
  access_token: string
  user: User
}

export function impersonateClient(clientId: string) {
  return apiClient.post<ImpersonateResponse>(`/admin/clients/${clientId}/impersonate`).then((r) => r.data)
}
