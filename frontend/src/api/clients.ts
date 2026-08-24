import { apiClient } from './client'
import type { ClientWithLicense, DurationType, License, User } from '@/types'

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
