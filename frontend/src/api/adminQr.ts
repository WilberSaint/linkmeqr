import { apiClient } from './client'
import type { QrCode } from '@/types'
import type { QrValidation } from './qr'

// Admin-scoped mirror of api/qr.ts — LinkMeQR Studio styles a client's QR on
// their behalf, reading/writing the same qr_codes row their own /me/qr
// editor would (so the style also applies everywhere else that QR is used:
// their profile, their loyalty card, and every print card).

// Owned by the client (not the admin) so it's indistinguishable from a logo
// the client uploaded themselves through their own /me/qr editor.
export function uploadClientMedia(clientId: string, file: File) {
  const form = new FormData()
  form.append('file', file)
  return apiClient
    .post<{ id: string; file_path: string }>(`/admin/clients/${clientId}/media/upload`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data)
}

export function getClientQr(clientId: string) {
  return apiClient.get<QrCode>(`/admin/clients/${clientId}/qr`).then((r) => r.data)
}

export function updateClientQr(
  clientId: string,
  payload: {
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
  },
) {
  return apiClient.patch<QrCode>(`/admin/clients/${clientId}/qr`, payload).then((r) => r.data)
}

export function validateClientQr(clientId: string) {
  return apiClient.get<QrValidation>(`/admin/clients/${clientId}/qr/validate`).then((r) => r.data)
}

export async function fetchClientQrPreview(clientId: string): Promise<string> {
  const res = await apiClient.get(`/admin/clients/${clientId}/qr/export`, {
    params: { format: 'png' },
    responseType: 'blob',
  })
  return URL.createObjectURL(res.data as Blob)
}

export async function downloadClientQrExport(clientId: string, format: 'png' | 'svg'): Promise<void> {
  const res = await apiClient.get(`/admin/clients/${clientId}/qr/export`, {
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
