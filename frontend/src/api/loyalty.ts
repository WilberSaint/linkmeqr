import { apiClient } from './client'
import type { ProfileTheme } from '@/types'

export interface LoyaltyProgram {
  id: string
  user_id: string
  stamps_required: number
  mid_reward_stamps: number | null
  mid_reward_description: string | null
  reward_description: string | null
  loyalty_token: string
  is_active: boolean
}

export interface LoyaltyCustomer {
  id: string
  user_id: string
  full_name: string
  phone: string | null
  stamps_count: number
  created_at: string
  updated_at: string
}

export interface LoyaltyCardStatus {
  needs_registration: boolean
  business_name: string
  theme: ProfileTheme | null
  logo_url: string | null
  full_name?: string
  stamps_count?: number
  stamps_required: number
  mid_reward_stamps: number | null
  mid_reward_description: string | null
  reward_description: string | null
  just_stamped?: boolean
  is_active: boolean
}

export interface WalletSaveInfo {
  enabled: boolean
  save_url?: string
  reason?: 'not_registered' | 'no_logo'
}

// --- Public (NFC tap flow, no auth) ---

export function getCardStatus(token: string) {
  return apiClient.get<LoyaltyCardStatus>(`/public/loyalty/${token}`).then((r) => r.data)
}

export function registerCard(token: string, payload: { full_name: string; phone?: string }) {
  return apiClient.post<LoyaltyCardStatus>(`/public/loyalty/${token}/register`, payload).then((r) => r.data)
}

// Tells the caller whether "Agregar a Google Wallet" should be shown at all
// (feature not configured / customer not registered / business has no logo
// are all normal, non-error states here — see reason).
export function getWalletSaveInfo(token: string) {
  return apiClient.get<WalletSaveInfo>(`/public/loyalty/${token}/wallet`).then((r) => r.data)
}

// --- Client (business owner) ---

export function getMyProgram() {
  return apiClient.get<LoyaltyProgram>('/me/loyalty').then((r) => r.data)
}

export function updateMyProgram(payload: {
  stamps_required: number
  mid_reward_stamps?: number | null
  mid_reward_description?: string | null
  reward_description?: string | null
  is_active: boolean
  regenerate_token?: boolean
}) {
  return apiClient.patch<LoyaltyProgram>('/me/loyalty', payload).then((r) => r.data)
}

export function listMyLoyaltyCustomers() {
  return apiClient.get<LoyaltyCustomer[]>('/me/loyalty/customers').then((r) => r.data)
}

export function stampCustomer(id: string) {
  return apiClient.post(`/me/loyalty/customers/${id}/stamp`)
}

export function redeemCustomer(id: string) {
  return apiClient.post(`/me/loyalty/customers/${id}/redeem`)
}

// Same blob-preview pattern as frontend/src/api/qr.ts — the export endpoint
// requires the Authorization header, so it can't be used as a plain <img src>.
export async function fetchLoyaltyQrPreview(): Promise<string> {
  const res = await apiClient.get('/me/loyalty/qr', { params: { format: 'png' }, responseType: 'blob' })
  return URL.createObjectURL(res.data as Blob)
}

export async function downloadLoyaltyQrExport(format: 'png' | 'svg'): Promise<void> {
  const res = await apiClient.get('/me/loyalty/qr', {
    params: { format, download: 1 },
    responseType: 'blob',
  })
  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `lealtad-qr.${format}`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
