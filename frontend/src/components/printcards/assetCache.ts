import { reactive } from 'vue'

import { apiClient } from '@/api/client'
import type { CardElement } from '@/types/cardLayout'
import type { QrTargetType } from '@/api/printCards'
import { captionRenderSegments } from './captionSegments'

/**
 * The editor draws the card itself so dragging is immediate, but two kinds of
 * artwork can only come from the backend: QR codes (which need the real
 * destination URL and the business's saved QR styling) and the built-in
 * glyphs (served from the exporter's own drawing code, so the icon on screen
 * is the icon that prints).
 *
 * Both are fetched once and cached for the session. Without the cache, every
 * drag frame that touched a QR element would re-request a code that cannot
 * have changed.
 */

export function iconCacheKey(name: string, color: string): string {
  return `${name}|${color}`
}

function qrCacheKey(clientId: string, targetType: string, targetValue?: string | null): string {
  return `${clientId}|${targetType}|${targetValue ?? ''}`
}

const qrCache = new Map<string, Promise<string>>()
const iconCache = new Map<string, Promise<string>>()

/**
 * Drops every cached QR. Restyling the business's shared QR changes the
 * artwork for a destination that has not itself changed, so the cache key
 * (which is the destination) cannot detect it — the caller has to say so.
 */
export function invalidateQrCache() {
  qrCache.clear()
}

async function fetchQr(clientId: string, targetType: QrTargetType, targetValue?: string | null): Promise<string> {
  const res = await apiClient.post(
    `/admin/clients/${clientId}/print-cards/qr-preview`,
    { target_type: targetType, target_value: targetValue || null },
    { responseType: 'text' },
  )
  return res.data as string
}

async function fetchIcon(name: string, color: string): Promise<string> {
  const res = await apiClient.get(`/admin/print-cards/icons/${encodeURIComponent(name)}`, {
    params: { color },
    responseType: 'text',
  })
  return res.data as string
}

/**
 * Resolves every asset the given elements reference into two reactive maps
 * the renderer reads. Returns immediately; the maps fill in as requests land,
 * so the canvas draws its placeholders first and swaps in real artwork
 * without blocking editing.
 */
export function useCardAssets(clientId: string) {
  /** Keyed by element id: a QR is per-element because each has its own destination. */
  const qrSvgs = reactive<Record<string, string>>({})
  /** Keyed by name+color: glyphs are shared across every element using them. */
  const iconSvgs = reactive<Record<string, string>>({})

  /** Requests one glyph if it is not already cached or in flight. */
  function wantIcon(name: string, color: string) {
    const key = iconCacheKey(name, color)
    if (iconSvgs[key]) return
    let pending = iconCache.get(key)
    if (!pending) {
      pending = fetchIcon(name, color)
      iconCache.set(key, pending)
      pending.catch(() => iconCache.delete(key))
    }
    pending
      .then((svg) => {
        iconSvgs[key] = svg
      })
      .catch(() => {
        /* the renderer shows its placeholder box */
      })
  }

  function resolve(elements: CardElement[]) {
    for (const el of elements) {
      if (el.hidden) continue

      if (el.type === 'qr') {
        const targetType = el.props.target_type
        if (!targetType) continue
        const key = qrCacheKey(clientId, targetType, el.props.target_value)
        let pending = qrCache.get(key)
        if (!pending) {
          pending = fetchQr(clientId, targetType, el.props.target_value)
          qrCache.set(key, pending)
          // A failed fetch must not poison the cache forever — a QR whose
          // block was momentarily missing should resolve on the next edit.
          pending.catch(() => qrCache.delete(key))
        }
        pending.then((svg) => {
          qrSvgs[el.id] = svg
        }).catch(() => {
          delete qrSvgs[el.id]
        })
      }

      if (el.type === 'icon') {
        wantIcon(el.props.name || 'star', el.props.color || '#111827')
      }

      // Toca/Escanea and any other "[icon] label" prompt draw the same
      // built-in glyphs an icon element does, in the prompt's own color.
      if (el.type === 'prompt' && el.props.icon) {
        wantIcon(el.props.icon, el.props.color || '#111827')
      }

      // A QR from before Toca/Escanea became independent elements can still
      // have a legacy caption embedded in its own props — whichever icons
      // the resolver says are actually showing have to be requested too, or
      // that caption renders with a gap where its icon belongs. Shared with
      // ElementRenderer so "which icon" can never disagree between what was
      // fetched and what gets drawn.
      if (el.type === 'qr') {
        const fill = el.props.caption_fill || '#111827'
        for (const seg of captionRenderSegments(el.props)) {
          if (seg.icon) wantIcon(seg.icon, fill)
        }
      }
    }
  }

  /** Forgets the cached codes and re-requests them for the given elements. */
  function refetchQrs(elements: CardElement[]) {
    invalidateQrCache()
    for (const key of Object.keys(qrSvgs)) delete qrSvgs[key]
    resolve(elements)
  }

  return { qrSvgs, iconSvgs, resolve, refetchQrs }
}
