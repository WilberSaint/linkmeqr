import { apiClient } from './client'
import type { ProfileTheme, Template } from '@/types'

export function listTemplates() {
  return apiClient.get<Template[]>('/templates').then((r) => r.data)
}

export function adminListTemplates() {
  return apiClient.get<Template[]>('/admin/templates/').then((r) => r.data)
}

export function getTemplate(id: string) {
  return apiClient.get<Template>(`/admin/templates/${id}`).then((r) => r.data)
}

export interface TemplatePayload {
  slug: string
  name: string
  description?: string | null
  default_theme: ProfileTheme
  sort_order?: number
}

export function createTemplate(payload: TemplatePayload) {
  return apiClient.post<Template>('/admin/templates/', payload).then((r) => r.data)
}

export function updateTemplate(id: string, payload: TemplatePayload) {
  return apiClient.patch<Template>(`/admin/templates/${id}`, payload).then((r) => r.data)
}

export function deleteTemplate(id: string) {
  return apiClient.delete(`/admin/templates/${id}`)
}

export function setTemplateActive(id: string, active: boolean) {
  return apiClient.post(`/admin/templates/${id}/${active ? 'activate' : 'deactivate'}`)
}
