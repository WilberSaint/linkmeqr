import { apiClient } from './client'
import type { BlockType } from '@/types'
import type { CardLayout, CardLayoutResponse, CardLayoutRevision } from '@/types/cardLayout'

export type PrintCardLayoutKey = 'google_review' | 'social_follow' | 'menu_scan' | 'loyalty_card' | 'multi_qr' | 'thank_you'
export type PrintCardSizePreset = 'business_card' | 'table_tent' | 'sticker_square' | 'door_hanger' | 'custom'
export type QrTargetType = 'profile' | 'menu' | 'loyalty' | 'block' | 'custom_url'
export type PrintCardSaleStatus = 'draft' | 'printed' | 'delivered'

export type PrintCardPattern = 'none' | 'dots' | 'lines' | 'grid' | 'waves' | 'circles'
export type PrintCardStyle = 'block' | 'split' | 'corners' | 'framed' | 'banner' | 'spotlight' | 'diagonal' | 'outline' | 'pattern'

export interface PrintCardColorOverrides {
  background?: string
  accent?: string
  text?: string
  pattern?: PrintCardPattern
  style?: PrintCardStyle
}

export type PrintCardTopIcon = 'platform' | 'logo'

export interface PrintCardContent {
  headline?: string
  subheadline?: string
  platform?: string
  left_label?: string
  right_label?: string
  left_target_type?: QrTargetType
  left_target_value?: string
  right_target_type?: QrTargetType
  right_target_value?: string
  discount_code?: string
  discount_label?: string
  top_icon?: PrintCardTopIcon
}

export interface PrintCard {
  id: string
  layout_key: PrintCardLayoutKey
  title: string | null
  size_preset: PrintCardSizePreset
  // Only set (both) when size_preset === 'custom'.
  custom_width_cm: number | null
  custom_height_cm: number | null
  qr_target_type: QrTargetType
  qr_target_value: string | null
  color_overrides: PrintCardColorOverrides | null
  content: PrintCardContent
  status: PrintCardSaleStatus
  sale_note: string | null
  scan_count: number
  created_at: string
  updated_at: string
}

export interface PrintCardPayload {
  layout_key: PrintCardLayoutKey
  title?: string | null
  size_preset: PrintCardSizePreset
  custom_width_cm?: number | null
  custom_height_cm?: number | null
  qr_target_type: QrTargetType
  qr_target_value?: string | null
  color_overrides?: PrintCardColorOverrides | null
  content: PrintCardContent
}

// A destination the QR can point at for one specific client — only the
// ones that actually exist for them (their profile/loyalty card always,
// plus one entry per social/menu/etc. block they actually have).
export interface QrTargetOption {
  target_type: QrTargetType
  target_value?: string // block id, when target_type === 'block'
  block_type?: BlockType
  title?: string | null
}

export function listPrintCards(clientId: string) {
  return apiClient.get<PrintCard[]>(`/admin/clients/${clientId}/print-cards`).then((r) => r.data)
}

export function getQrTargets(clientId: string) {
  return apiClient.get<QrTargetOption[]>(`/admin/clients/${clientId}/print-cards/qr-targets`).then((r) => r.data)
}

export function createPrintCard(clientId: string, payload: PrintCardPayload) {
  return apiClient.post<PrintCard>(`/admin/clients/${clientId}/print-cards`, payload).then((r) => r.data)
}

export function getPrintCard(clientId: string, id: string) {
  return apiClient.get<PrintCard>(`/admin/clients/${clientId}/print-cards/${id}`).then((r) => r.data)
}

export function updatePrintCard(clientId: string, id: string, payload: PrintCardPayload) {
  return apiClient.patch<PrintCard>(`/admin/clients/${clientId}/print-cards/${id}`, payload).then((r) => r.data)
}

export function deletePrintCard(clientId: string, id: string) {
  return apiClient.delete(`/admin/clients/${clientId}/print-cards/${id}`)
}

export function updatePrintCardStatus(clientId: string, id: string, status: PrintCardSaleStatus, saleNote: string | null) {
  return apiClient
    .patch<PrintCard>(`/admin/clients/${clientId}/print-cards/${id}/status`, { status, sale_note: saleNote })
    .then((r) => r.data)
}

export const PRINT_CARD_SALE_STATUSES: { value: PrintCardSaleStatus; label: string }[] = [
  { value: 'draft', label: 'Borrador' },
  { value: 'printed', label: 'Impresa' },
  { value: 'delivered', label: 'Entregada' },
]

// Stateless preview: renders SVG without saving anything. Passing a payload
// renders one of the built-in designs (the layout picker's thumbnails);
// passing a tree renders exactly that tree, which is how the editor shows the
// real exported output while the designer is still dragging things around.
export async function previewPrintCard(clientId: string, payload: PrintCardPayload): Promise<string> {
  const res = await apiClient.post(`/admin/clients/${clientId}/print-cards/preview`, payload, { responseType: 'text' })
  return res.data as string
}

export async function previewLayout(clientId: string, layout: CardLayout): Promise<string> {
  const res = await apiClient.post(`/admin/clients/${clientId}/print-cards/preview`, { layout }, { responseType: 'text' })
  return res.data as string
}

// --- Element tree -----------------------------------------------------------

export function getCardLayout(clientId: string, cardId: string) {
  return apiClient.get<CardLayoutResponse>(`/admin/clients/${clientId}/print-cards/${cardId}/layout`).then((r) => r.data)
}

// baseVersion is the revision the editor loaded. The backend rejects a save
// built on a stale one (409 layout_stale) rather than silently discarding
// whatever another admin saved in the meantime.
export function saveCardLayout(clientId: string, cardId: string, layout: CardLayout, baseVersion: number | null) {
  return apiClient
    .put<CardLayoutResponse>(`/admin/clients/${clientId}/print-cards/${cardId}/layout`, { layout, base_version: baseVersion })
    .then((r) => r.data)
}

export function listCardLayoutVersions(clientId: string, cardId: string) {
  return apiClient
    .get<CardLayoutRevision[]>(`/admin/clients/${clientId}/print-cards/${cardId}/layout/versions`)
    .then((r) => r.data)
}

export function restoreCardLayoutVersion(clientId: string, cardId: string, version: number) {
  return apiClient
    .post<CardLayoutResponse>(`/admin/clients/${clientId}/print-cards/${cardId}/layout/versions/${version}/restore`)
    .then((r) => r.data)
}

// Hands back a fresh tree for one of the built-in designs without creating a
// card — this is what makes the six templates a starting point rather than a
// rendering mode. Once seeded, the editor owns the tree outright.
export function seedCardLayout(clientId: string, payload: PrintCardPayload) {
  return apiClient
    .post<CardLayoutResponse>(`/admin/clients/${clientId}/print-cards/seed-layout`, payload)
    .then((r) => r.data)
}

export async function getPrintCardSvg(clientId: string, id: string): Promise<string> {
  const res = await apiClient.get(`/admin/clients/${clientId}/print-cards/${id}/export`, { responseType: 'text' })
  return res.data as string
}

export function downloadSvgText(svg: string, filename: string) {
  const blob = new Blob([svg], { type: 'image/svg+xml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

// Rasterizes an SVG string to a PNG blob in the browser (300 DPI at the
// card's own physical size), then triggers a download. Done client-side —
// the backend only ever needs to produce the SVG, which is DPI-independent.
export async function downloadSvgAsPng(svg: string, filename: string, widthIn: number, heightIn: number, dpi = 300) {
  const widthPx = Math.round(widthIn * dpi)
  const heightPx = Math.round(heightIn * dpi)

  const svgBlob = new Blob([svg], { type: 'image/svg+xml' })
  const svgUrl = URL.createObjectURL(svgBlob)
  try {
    const img = new Image()
    await new Promise<void>((resolve, reject) => {
      img.onload = () => resolve()
      img.onerror = () => reject(new Error('No se pudo cargar el SVG para convertirlo a PNG.'))
      img.src = svgUrl
    })

    const canvas = document.createElement('canvas')
    canvas.width = widthPx
    canvas.height = heightPx
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Canvas no disponible.')
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, widthPx, heightPx)
    ctx.drawImage(img, 0, 0, widthPx, heightPx)

    const pngBlob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
    if (!pngBlob) throw new Error('No se pudo generar el PNG.')

    const pngUrl = URL.createObjectURL(pngBlob)
    const a = document.createElement('a')
    a.href = pngUrl
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(pngUrl)
  } finally {
    URL.revokeObjectURL(svgUrl)
  }
}

// widthIn/heightIn stay the internal source of truth (matches the backend's
// SizePresets, and downloadSvgAsPng's DPI math) — only the labels display in
// centimeters, the unit that actually means something to a local print shop.
export const PRINT_CARD_SIZES: Record<PrintCardSizePreset, { label: string; widthIn: number; heightIn: number }> = {
  business_card: { label: 'Tarjeta (8.9 × 5.1 cm)', widthIn: 3.5, heightIn: 2 },
  table_tent: { label: 'Exhibidor de mesa (10.2 × 15.2 cm)', widthIn: 4, heightIn: 6 },
  sticker_square: { label: 'Sticker cuadrado (5.1 × 5.1 cm)', widthIn: 2, heightIn: 2 },
  door_hanger: { label: 'Colgante de puerta (8.9 × 21.6 cm)', widthIn: 3.5, heightIn: 8.5 },
  custom: { label: 'Personalizado', widthIn: 3.5, heightIn: 2 },
}

// The visual treatment applied on top of a design's arrangement. Independent
// of the arrangement itself, so any design can be rendered in any style.
export const PRINT_CARD_STYLES: { value: PrintCardStyle; label: string; description: string }[] = [
  { value: 'block', label: 'Bloque de color', description: 'Tu color de marca de borde a borde.' },
  { value: 'split', label: 'Dos zonas', description: 'Franja de color arriba, zona blanca con el QR abajo.' },
  { value: 'corners', label: 'Esquinas', description: 'Tarjeta oscura con acentos triangulares.' },
  { value: 'framed', label: 'Marco', description: 'Borde de color e interior claro.' },
  { value: 'banner', label: 'Franja superior', description: 'Banda de color arriba y QR grande sobre blanco.' },
  { value: 'spotlight', label: 'Foco oscuro', description: 'Tarjeta oscura, estrellas grandes, QR muy visible.' },
  { value: 'diagonal', label: 'Cuña diagonal', description: 'Como «Franja superior», pero cortada en ángulo — más dinámica.' },
  { value: 'outline', label: 'Minimalista', description: 'Solo una línea delgada de tu color. Todo el resto es espacio en blanco.' },
  { value: 'pattern', label: 'Con textura', description: 'Tu color de marca con una textura sutil de fondo, y el contenido en un panel blanco.' },
]

export const PRINT_CARD_LAYOUTS: { key: PrintCardLayoutKey; label: string; description: string }[] = [
  { key: 'google_review', label: 'Reseña de Google', description: 'Estrellas + QR a tu link de reseña.' },
  { key: 'social_follow', label: 'Síguenos en redes', description: 'Ícono de tu red + QR a tu perfil o red social.' },
  { key: 'menu_scan', label: 'Escanea el menú', description: 'QR directo a tu menú, sin pasar por el perfil.' },
  { key: 'loyalty_card', label: 'Tarjeta de lealtad', description: 'QR al enlace para sellar tu tarjeta de sellos.' },
  { key: 'multi_qr', label: 'Combinada (2 QR)', description: 'Dos secciones lado a lado, cada una con su propio QR.' },
  { key: 'thank_you', label: 'Gracias por tu compra', description: 'Mensaje de agradecimiento, código de descuento opcional, y QR de reseña.' },
]

// Exports the card as a vector PDF, at its exact physical size. Done in the
// browser on the same SVG the backend produced, for the same reason the PNG
// path already is: the SVG is the single rendered artifact, and converting it
// here keeps the vector intact end to end — a print shop gets real curves and
// selectable text rather than a rasterized page.
export async function downloadSvgAsPdf(svg: string, filename: string, widthIn: number, heightIn: number) {
  const { jsPDF } = await import('jspdf')
  const { svg2pdf } = await import('svg2pdf.js')

  const doc = new jsPDF({
    unit: 'in',
    format: [widthIn, heightIn],
    orientation: widthIn > heightIn ? 'landscape' : 'portrait',
  })

  // svg2pdf measures text and shapes through the live DOM, so the element has
  // to be attached to be measurable. It is parked off-screen rather than
  // display:none, which would collapse every box to zero.
  const holder = document.createElement('div')
  holder.style.cssText = 'position:fixed;left:-10000px;top:0;width:0;height:0;overflow:hidden'
  holder.innerHTML = svg
  const svgEl = holder.firstElementChild as SVGSVGElement | null
  if (!svgEl) throw new Error('No se pudo interpretar el SVG de la tarjeta.')
  document.body.appendChild(holder)

  try {
    await svg2pdf(svgEl, doc, { width: widthIn, height: heightIn })
    doc.save(filename)
  } finally {
    holder.remove()
  }
}

// The physical size of a card, in inches — the unit every export path needs
// and the one PRINT_CARD_SIZES stores. A custom size carries centimetres
// because that is what a local print shop asks for.
export function cardSizeInches(card: Pick<PrintCard, 'size_preset' | 'custom_width_cm' | 'custom_height_cm'>): {
  widthIn: number
  heightIn: number
} {
  if (card.size_preset === 'custom' && card.custom_width_cm && card.custom_height_cm) {
    return { widthIn: card.custom_width_cm / 2.54, heightIn: card.custom_height_cm / 2.54 }
  }
  const preset = PRINT_CARD_SIZES[card.size_preset] ?? PRINT_CARD_SIZES.business_card
  return { widthIn: preset.widthIn, heightIn: preset.heightIn }
}
