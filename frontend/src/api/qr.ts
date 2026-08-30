import { apiClient } from './client'
import type { QrCode } from '@/types'

export function getMyQr() {
  return apiClient.get<QrCode>('/me/qr').then((r) => r.data)
}

export function updateMyQr(payload: {
  foreground_color: string
  background_color: string
  module_style: string
  eye_style: string
  logo_media_id?: string | null
  logo_style?: string
  eye_color_from_logo?: boolean
  preset_icon?: string | null
  frame_shape?: string | null
  shape_fill?: boolean
}) {
  return apiClient.patch<QrCode>('/me/qr', payload).then((r) => r.data)
}

export interface QrValidation {
  warnings: string[]
  effective_error_correction: string
  contrast_ratio: number
}

export function validateMyQr() {
  return apiClient.get<QrValidation>('/me/qr/validate').then((r) => r.data)
}

// The export endpoint requires the Authorization header, so it can't be used
// as a plain <img src>. Fetch it as a blob and hand back an object URL that
// the caller must revoke (URL.revokeObjectURL) when no longer needed.
export async function fetchQrPreview(): Promise<string> {
  const res = await apiClient.get('/me/qr/export', { params: { format: 'png' }, responseType: 'blob' })
  return URL.createObjectURL(res.data as Blob)
}

export async function downloadQrExport(format: 'png' | 'svg'): Promise<void> {
  const res = await apiClient.get('/me/qr/export', {
    params: { format, download: 1 },
    responseType: 'blob',
  })
  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `qr.${format}`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
