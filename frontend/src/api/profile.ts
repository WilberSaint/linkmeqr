import { apiClient } from './client'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'

export function getMyProfile() {
  return apiClient.get<Profile>('/me/profile').then((r) => r.data)
}

export function updateMyProfile(payload: {
  business_name: string
  description?: string | null
  template_id?: string | null
  logo_media_id?: string | null
  cover_media_id?: string | null
  is_published: boolean
}) {
  return apiClient.patch<Profile>('/me/profile', payload).then((r) => r.data)
}

export function getMyTheme() {
  return apiClient.get<ProfileTheme>('/me/theme').then((r) => r.data)
}

export function updateMyTheme(payload: ProfileTheme) {
  return apiClient.patch<ProfileTheme>('/me/theme', payload).then((r) => r.data)
}

export function listMyBlocks() {
  return apiClient.get<ProfileBlock[]>('/me/blocks').then((r) => r.data)
}

export function createBlock(payload: Partial<ProfileBlock>) {
  return apiClient.post<ProfileBlock>('/me/blocks', payload).then((r) => r.data)
}

export function updateBlock(id: string, payload: Partial<ProfileBlock>) {
  return apiClient.patch<ProfileBlock>(`/me/blocks/${id}`, payload).then((r) => r.data)
}

export function deleteBlock(id: string) {
  return apiClient.delete(`/me/blocks/${id}`)
}

export function duplicateBlock(id: string) {
  return apiClient.post<ProfileBlock>(`/me/blocks/${id}/duplicate`).then((r) => r.data)
}

export function reorderBlocks(items: { id: string; sort_order: number }[]) {
  return apiClient.patch('/me/blocks/reorder', { items })
}

export function uploadMedia(file: File) {
  const form = new FormData()
  form.append('file', file)
  return apiClient
    .post<{ id: string; file_path: string }>('/media/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data)
}
