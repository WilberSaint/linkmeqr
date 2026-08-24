import axios from 'axios'
import type { PublicProfileResponse } from '@/types'

// Public endpoints never send the auth token, so they use a bare axios
// instance rather than the interceptor-wrapped apiClient.
const publicClient = axios.create({ baseURL: '/api' })

export function getPublicProfile(slug: string) {
  return publicClient.get<PublicProfileResponse>(`/public/profiles/${slug}`).then((r) => r.data)
}

export function trackEvent(slug: string, type: 'VIEW' | 'BLOCK_CLICK', blockId?: string) {
  return publicClient.post(`/public/profiles/${slug}/events`, { type, block_id: blockId })
}
