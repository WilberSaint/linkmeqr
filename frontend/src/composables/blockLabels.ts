import type { BlockType } from '@/types'

export const BLOCK_LABELS: Record<BlockType, string> = {
  instagram: 'Instagram',
  facebook: 'Facebook',
  tiktok: 'TikTok',
  youtube: 'YouTube',
  whatsapp: 'WhatsApp',
  phone: 'Teléfono',
  email: 'Email',
  location: 'Ubicación',
  website: 'Sitio web',
  menu: 'Menú',
  catalog: 'Catálogo',
  image: 'Imagen',
  video: 'Video',
  text: 'Texto',
  link: 'Enlace personalizado',
  google_review: 'Reseña en Google',
  gallery: 'Galería de fotos',
  hours: 'Horario de atención',
  testimonials: 'Testimonios',
  map: 'Mapa',
}

export function blockLabel(type: BlockType): string {
  return BLOCK_LABELS[type] ?? type
}
