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
}

export function blockLabel(type: BlockType): string {
  return BLOCK_LABELS[type] ?? type
}
