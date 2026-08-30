// The print-card element tree. This mirrors models.CardLayout in the Go
// backend field-for-field — the same document is stored, rendered by the
// exporter, and edited here, so any change on one side needs the same change
// on the other.

import type { QrTargetType } from '@/api/printCards'

/** Every x/y/w/h below is in hundredths of a physical inch. */
export const CARD_UNITS_PER_INCH = 100

export type ElementType = 'text' | 'image' | 'qr' | 'shape' | 'icon' | 'prompt'

export interface TextProps {
  text: string
  font_family: string
  font_size: number
  font_weight: number
  color: string
  align: 'left' | 'center' | 'right'
  line_height: number
  letter_spacing: number
  italic: boolean
  underline: boolean
  uppercase: boolean
}

export interface ImageProps {
  source: 'media' | 'logo'
  media_id?: string
  /**
   * Denormalized alongside media_id, the same way ProfileBlock carries
   * media_url next to media_id — the editor can then render the picture
   * straight from the tree, with no extra fetch to resolve an id before
   * anything shows up. The backend still resolves media_id itself at export
   * time (ResolveLayoutAssets), independent of whatever this holds.
   */
  media_url?: string
  fit: 'cover' | 'contain'
  shape: 'circle' | 'rounded' | 'square' | ''
  radius: number
}

export interface QrProps {
  target_type: QrTargetType
  target_value?: string | null
  /**
   * The /q/:code/:slot segment this QR's earlier exports already printed.
   * Never edit or clear it: physical cards in customers' hands encode it,
   * and the redirect resolves them by matching against it.
   */
  legacy_slot?: string
  frame: 'none' | 'square' | 'rounded'
  frame_fill: string
  frame_radius: number
  frame_pad: number
  shadow: boolean

  // --- legacy embedded caption ----------------------------------------
  //
  // Toca and Escanea used to live baked into the QR's own props; they are
  // independent 'prompt' elements now (see PromptProps), so a QR added from
  // today onward never sets any of the fields below. They stay on the type
  // only so a QR saved before that split — real cards printed under the old
  // scheme — keeps rendering exactly as it did; captionSegments.ts is the
  // one place that still reads them. Never write to them from new code.
  caption_mode?: 'icons' | 'text' | 'none'
  show_tap?: boolean
  tap_icon?: string
  tap_text?: string
  show_scan?: boolean
  scan_icon?: string
  scan_text?: string
  caption?: 'none' | 'dual' | 'tap' | 'scan' | 'scan_me' | 'text'
  caption_text?: string
  caption_size?: number
  caption_fill?: string
  caption_bare?: boolean
}

/**
 * One "[icon] label" call-to-action — "Toca", "Escanea", or any other short
 * instruction paired with a glyph. Its own independent element, freely
 * positioned, resized and restyled apart from whatever QR (or nothing at
 * all) it happens to sit near.
 */
export interface PromptProps {
  /** An iconGlyph name — the "tap" family (contactless | tap_hand | tap_card), the "scan" family (scan | scan_qr | scan_camera), or any other. Empty draws text only. */
  icon?: string
  text: string
  color: string
  /** Drops the pill behind the prompt, leaving just the icon and text — the default, and the only style shipped so far. */
  bare: boolean
  /** Overrides the size the element's own box height would otherwise derive. 0 means "size to the box". */
  font_size?: number
}

export interface ShapeProps {
  kind: 'rect' | 'ellipse' | 'line' | 'polygon' | 'path'
  fill: string
  stroke: string
  stroke_width: number
  dash_array?: string
  radius: number
  /** Local box coordinates (0..w, 0..h), so they move and scale with the element. */
  points?: string
  path?: string
  shadow: boolean
  /** One of the card background's own textures, layered over fill rather than replacing it. Empty means a plain solid fill. */
  pattern?: '' | 'dots' | 'lines' | 'grid' | 'waves' | 'circles'
  /** Empty defaults to fill. */
  pattern_ink?: string
}

export interface IconProps {
  name: string
  color: string
  /** Above 1, the glyph is laid out in a row (the rating stars). */
  count: number
  gap: number
}

export type ElementProps = Partial<TextProps & ImageProps & QrProps & ShapeProps & IconProps & PromptProps>

export interface CardElement {
  id: string
  type: ElementType
  x: number
  y: number
  w: number
  h: number
  /** Degrees clockwise, about the element's own center. */
  rotation: number
  /** Paint order, lowest first. The store keeps this dense and in sync with the array. */
  z_index: number
  hidden: boolean
  locked: boolean
  name?: string
  opacity?: number
  props: ElementProps
}

export interface CardBackground {
  fill: string
  gradient_to?: string
  pattern?: '' | 'dots' | 'lines' | 'grid' | 'waves' | 'circles'
  pattern_ink?: string
}

export interface DieCut {
  kind: 'circle'
  cx: number
  cy: number
  r: number
}

export interface CardCanvas {
  w: number
  h: number
  corner_r: number
  die_cut?: DieCut | null
}

export interface CardLayout {
  version: number
  canvas: CardCanvas
  background: CardBackground
  elements: CardElement[]
  seeded_from?: string
}

export interface CardLayoutResponse {
  layout: CardLayout
  layout_version: number
}

export interface CardLayoutRevision {
  id: string
  print_card_id: string
  version: number
  created_by: string | null
  created_at: string
}

/** A prompt's flavour picks which glyph+wording it starts with — used only when adding one fresh. */
export type PromptVariant = 'tap' | 'scan'

/**
 * Default props for a newly added element of each type. Kept here rather than
 * in the component that creates them so "add a text box" behaves identically
 * whether it comes from the toolbar, a paste, or a duplicate.
 */
export function defaultProps(type: ElementType, variant?: PromptVariant): ElementProps {
  switch (type) {
    case 'text':
      return {
        text: 'Texto',
        font_family: 'Arial, sans-serif',
        font_size: 18,
        font_weight: 700,
        color: '#111827',
        align: 'center',
        line_height: 1.2,
        letter_spacing: 0,
        italic: false,
        underline: false,
        uppercase: false,
      }
    case 'image':
      return { source: 'logo', fit: 'cover', shape: 'circle', radius: 0 }
    case 'qr':
      // No embedded caption: Toca/Escanea are their own elements now, added
      // alongside a QR rather than baked into it — see the toolbar's "+ QR"
      // action, which adds this plus a matching prompt pair in one gesture.
      return {
        target_type: 'profile',
        frame: 'rounded',
        frame_fill: '#ffffff',
        frame_radius: 0.16,
        frame_pad: 0.18,
        shadow: true,
      }
    case 'shape':
      return { kind: 'rect', fill: '#ffffff', stroke: '', stroke_width: 0, radius: 0, shadow: false }
    case 'icon':
      return { name: 'star', color: '#111827', count: 1, gap: 1.3 }
    case 'prompt':
      return variant === 'scan'
        ? { icon: 'scan', text: 'Escanea', color: '#111827', bare: true }
        : { icon: 'contactless', text: 'Toca', color: '#111827', bare: true }
  }
}

/** Sensible starting box, in card units, for a newly added element. */
export function defaultSize(type: ElementType): { w: number; h: number } {
  switch (type) {
    case 'text':
      return { w: 180, h: 26 }
    case 'qr':
      return { w: 110, h: 110 }
    case 'image':
    case 'icon':
      return { w: 60, h: 60 }
    case 'shape':
      return { w: 100, h: 60 }
    case 'prompt':
      return { w: 70, h: 20 }
  }
}

const TYPE_LABELS: Record<ElementType, string> = {
  text: 'Texto',
  image: 'Imagen',
  qr: 'Código QR',
  shape: 'Forma',
  icon: 'Ícono',
  prompt: 'Llamada a la acción',
}

/**
 * What the layers panel shows for an element: its own name if the designer
 * set one, otherwise something derived from its content — a text layer reads
 * far better as its own words than as "Texto 3".
 */
export function elementLabel(el: CardElement): string {
  if (el.name) return el.name
  if (el.type === 'text' && el.props.text) {
    const firstLine = el.props.text.split('\n')[0]
    return firstLine.length > 24 ? `${firstLine.slice(0, 24)}…` : firstLine
  }
  if (el.type === 'prompt' && el.props.text) return el.props.text
  return TYPE_LABELS[el.type]
}
