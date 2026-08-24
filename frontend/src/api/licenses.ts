import { apiClient } from './client'
import type { ActivationCode, DurationType, License, LicenseActivation } from '@/types'

export function activateCode(code: string) {
  return apiClient.post<License>('/me/license/activate', { code }).then((r) => r.data)
}

export function myLicenseHistory() {
  return apiClient.get<LicenseActivation[]>('/me/license/history').then((r) => r.data)
}

export function generateCode(payload: { duration_type: DurationType; custom_days?: number }) {
  return apiClient.post<ActivationCode>('/admin/licenses/codes', payload).then((r) => r.data)
}

export function generateBatch(payload: { duration_type: DurationType; custom_days?: number; quantity: number }) {
  return apiClient.post<ActivationCode[]>('/admin/licenses/codes/batch', payload).then((r) => r.data)
}

export function listCodes(filter?: { status?: string; batch_id?: string }) {
  return apiClient
    .get<ActivationCode[]>('/admin/licenses/codes', { params: filter })
    .then((r) => r.data)
}

export function revokeCode(id: string) {
  return apiClient.post(`/admin/licenses/codes/${id}/revoke`)
}

export function clientLicenseHistory(userId: string) {
  return apiClient.get<LicenseActivation[]>(`/admin/licenses/${userId}/history`).then((r) => r.data)
}
