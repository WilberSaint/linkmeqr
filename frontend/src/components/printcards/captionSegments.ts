import type { ElementProps } from '@/types/cardLayout'

/**
 * Resolves a QR's caption configuration into segments — mirrors
 * captionSegments/legacyCaptionSegments in the Go renderer (print_card_render.go)
 * field for field. Used by the canvas (what to draw), the asset cache (which
 * icon artwork to fetch), and the properties panel (what the Toca/Escanea
 * toggles should show as checked) — one implementation instead of three, so
 * they cannot drift out of sync with each other or with the export.
 */

export interface CaptionSegment {
  icon: string
  label: string
}

export const DEFAULT_TAP_ICON = 'contactless'
export const DEFAULT_TAP_TEXT = 'Toca'
export const DEFAULT_SCAN_ICON = 'scan'
export const DEFAULT_SCAN_TEXT = 'Escanea'

/** The two independent prompts a QR's caption is built from, resolved to concrete values. */
export interface CaptionState {
  showTap: boolean
  tapIcon: string
  tapText: string
  showScan: boolean
  scanIcon: string
  scanText: string
}

/**
 * Derives the Toca/Escanea toggle state the properties panel shows.
 *
 * A QR saved before the two prompts became independent has no show_tap/
 * show_scan of its own — this reads its legacy `caption` string instead, so
 * the panel's checkboxes always reflect what is actually printing, never a
 * blank slate. `caption_mode: 'text'` (a bare custom line, no icon) and
 * `'none'` have no meaningful Toca/Escanea state, so both read as off; editing
 * either toggle on such an element migrates it to the icons format outright
 * (see PropertiesPanel's updateCaption), which is an acceptable one-way
 * conversion since no card in the wild uses the bare-text mode.
 */
export function resolveCaptionState(p: ElementProps): CaptionState {
  if (p.caption_mode === 'icons') {
    return {
      showTap: p.show_tap ?? false,
      tapIcon: p.tap_icon || DEFAULT_TAP_ICON,
      tapText: p.tap_text || DEFAULT_TAP_TEXT,
      showScan: p.show_scan ?? false,
      scanIcon: p.scan_icon || DEFAULT_SCAN_ICON,
      scanText: p.scan_text || DEFAULT_SCAN_TEXT,
    }
  }
  if (!p.caption_mode) {
    switch (p.caption) {
      case 'dual':
        return {
          showTap: true,
          tapIcon: DEFAULT_TAP_ICON,
          tapText: p.caption_text || DEFAULT_TAP_TEXT,
          showScan: true,
          scanIcon: DEFAULT_SCAN_ICON,
          scanText: DEFAULT_SCAN_TEXT,
        }
      case 'tap':
        return {
          showTap: true,
          tapIcon: DEFAULT_TAP_ICON,
          tapText: p.caption_text || DEFAULT_TAP_TEXT,
          showScan: false,
          scanIcon: DEFAULT_SCAN_ICON,
          scanText: DEFAULT_SCAN_TEXT,
        }
      case 'scan':
        return {
          showTap: false,
          tapIcon: DEFAULT_TAP_ICON,
          tapText: DEFAULT_TAP_TEXT,
          showScan: true,
          scanIcon: DEFAULT_SCAN_ICON,
          scanText: p.caption_text || DEFAULT_SCAN_TEXT,
        }
      case 'scan_me':
        return {
          showTap: false,
          tapIcon: DEFAULT_TAP_ICON,
          tapText: DEFAULT_TAP_TEXT,
          showScan: true,
          scanIcon: DEFAULT_SCAN_ICON,
          scanText: p.caption_text || 'Escanéame',
        }
    }
  }
  return {
    showTap: false,
    tapIcon: DEFAULT_TAP_ICON,
    tapText: DEFAULT_TAP_TEXT,
    showScan: false,
    scanIcon: DEFAULT_SCAN_ICON,
    scanText: DEFAULT_SCAN_TEXT,
  }
}

/** The ordered segments to actually draw. Empty means the caption is switched off. */
export function captionRenderSegments(p: ElementProps): CaptionSegment[] {
  if (!p.caption_mode && p.caption) {
    return legacySegments(p.caption, p.caption_text)
  }
  if (p.caption_mode === 'text') {
    return p.caption_text ? [{ icon: '', label: p.caption_text }] : []
  }
  if (p.caption_mode !== 'icons') return []

  const segments: CaptionSegment[] = []
  if (p.show_tap) segments.push({ icon: p.tap_icon || DEFAULT_TAP_ICON, label: p.tap_text || DEFAULT_TAP_TEXT })
  if (p.show_scan) segments.push({ icon: p.scan_icon || DEFAULT_SCAN_ICON, label: p.scan_text || DEFAULT_SCAN_TEXT })
  return segments
}

function legacySegments(style: string, custom?: string): CaptionSegment[] {
  if (style === 'text') return custom ? [{ icon: '', label: custom }] : []
  switch (style) {
    case 'dual':
      return [
        { icon: DEFAULT_TAP_ICON, label: custom || DEFAULT_TAP_TEXT },
        { icon: DEFAULT_SCAN_ICON, label: DEFAULT_SCAN_TEXT },
      ]
    case 'tap':
      return [{ icon: DEFAULT_TAP_ICON, label: custom || DEFAULT_TAP_TEXT }]
    case 'scan':
      return [{ icon: DEFAULT_SCAN_ICON, label: custom || DEFAULT_SCAN_TEXT }]
    case 'scan_me':
      return [{ icon: DEFAULT_SCAN_ICON, label: custom || 'Escanéame' }]
    default:
      return []
  }
}
