const loadedFonts = new Set<string>()

/**
 * Injects a Google Fonts <link> for the given family the first time it's
 * requested, so font-family choices in the theme editor/preview actually
 * render instead of silently falling back to the system font.
 */
export function ensureGoogleFontLoaded(family: string) {
  if (!family || loadedFonts.has(family)) return
  loadedFonts.add(family)

  const link = document.createElement('link')
  link.rel = 'stylesheet'
  const encoded = family.trim().replace(/\s+/g, '+')
  link.href = `https://fonts.googleapis.com/css2?family=${encoded}:wght@400;500;600;700&display=swap`
  document.head.appendChild(link)
}

export const AVAILABLE_FONTS = [
  'Inter',
  'Poppins',
  'Playfair Display',
  'Roboto',
  'Montserrat',
  'Lato',
  'Oswald',
  'Merriweather',
  'Nunito',
  'Raleway',
] as const
