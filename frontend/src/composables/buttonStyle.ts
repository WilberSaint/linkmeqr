import type { ProfileTheme } from '@/types'

/**
 * The corner treatment for theme.button_style, shared by every button on the
 * public page — link blocks, the Google review CTA, and save-contact/share —
 * so choosing "Píldora" in the editor actually means every button on the
 * page, not just the content blocks.
 */
export function buttonShapeClass(style: ProfileTheme['button_style'] | undefined): string {
  switch (style) {
    case 'square':
      return 'rounded-none'
    case 'pill':
      return 'rounded-full'
    case 'outline':
      return 'rounded-lg border-2 bg-transparent'
    default:
      return 'rounded-xl'
  }
}
