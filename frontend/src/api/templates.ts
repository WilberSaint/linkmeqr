import { apiClient } from './client'
import type { Template } from '@/types'

export function listTemplates() {
  return apiClient.get<Template[]>('/templates').then((r) => r.data)
}
