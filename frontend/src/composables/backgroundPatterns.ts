export type PatternDirection = 'horizontal' | 'vertical' | 'diagonal'

export interface PatternDef {
  id: string
  label: string
  /** SVG path/shape markup for one motif, drawn inside a 24x24 viewBox, using `currentColor`. */
  motif: string
}

// Simple single-color line-art motifs, drawn by hand as SVG paths (24x24 viewBox).
export const PATTERNS: PatternDef[] = [
  {
    id: 'coffee-bean',
    label: 'Café',
    motif:
      '<ellipse cx="12" cy="12" rx="7" ry="9" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="M12 3.5c-2.2 2.4-2.2 6.1 0 8.5s2.2 6.1 0 8.5" fill="none" stroke="currentColor" stroke-width="1.6"/>',
  },
  {
    id: 'matcha-leaf',
    label: 'Matcha',
    motif:
      '<path d="M12 21c-6-1-9-6-8-13 7-1 12 2 13 8 .6 3.4-1.6 5.6-5 5Z" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="M5 8c4 2 7 6 7 13" fill="none" stroke="currentColor" stroke-width="1.4"/>',
  },
  {
    id: 'heart',
    label: 'Corazones',
    motif:
      '<path d="M12 20.5S3.5 15 3.5 9a4.5 4.5 0 0 1 8.5-2 4.5 4.5 0 0 1 8.5 2c0 6-8.5 11.5-8.5 11.5Z" fill="none" stroke="currentColor" stroke-width="1.6"/>',
  },
  {
    id: 'clover',
    label: 'Suerte',
    motif:
      '<path d="M12 12c0-3 -2.5-5-5-3.2C5.6 10 6.6 12.6 9 13c-2.4.4-3.4 3-2 4.8C9.5 19.6 12 17.6 12 15c0 2.6 2.5 4.6 5 2.8 1.4-1.8.4-4.4-2-4.8 2.4-.4 3.4-3 2-4.8-2.5-1.8-5 .2-5 3.2Z" fill="none" stroke="currentColor" stroke-width="1.4"/>',
  },
  {
    id: 'star',
    label: 'Estrellas',
    motif:
      '<path d="m12 2 2.6 6.6L21 10l-5.2 4.3L17.4 21 12 17.3 6.6 21l1.6-6.7L3 10l6.4-1.4Z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/>',
  },
  {
    id: 'flower',
    label: 'Flores',
    motif:
      '<circle cx="12" cy="12" r="2.2" fill="none" stroke="currentColor" stroke-width="1.4"/><circle cx="12" cy="6" r="3" fill="none" stroke="currentColor" stroke-width="1.3"/><circle cx="12" cy="18" r="3" fill="none" stroke="currentColor" stroke-width="1.3"/><circle cx="6" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.3"/><circle cx="18" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.3"/>',
  },
  {
    id: 'dot',
    label: 'Puntos',
    motif: '<circle cx="12" cy="12" r="1.8" fill="currentColor"/>',
  },
  {
    id: 'cross-stitch',
    label: 'Líneas',
    motif: '<path d="M4 4 20 20M20 4 4 20" stroke="currentColor" stroke-width="1.4"/>',
  },
]

function motifSvg(motif: string, color: string): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" color="${color}">${motif}</svg>`
}

/**
 * Builds a CSS `background` shorthand value: a repeating pattern of the given
 * motif over a solid base color, oriented per `direction`. This value is
 * stored directly as ProfileTheme.background_value when background_type is
 * 'pattern' — the public preview just applies it as `background`.
 */
export function buildPatternBackground(
  patternId: string,
  baseColor: string,
  motifColor: string,
  direction: PatternDirection,
  spacing = 40,
): string {
  const pattern = PATTERNS.find((p) => p.id === patternId) ?? PATTERNS[0]
  const svg = motifSvg(pattern.motif, motifColor)
  const encoded = encodeURIComponent(svg).replace(/'/g, '%27').replace(/"/g, '%22')
  const dataUri = `url("data:image/svg+xml,${encoded}")`

  let position = '0 0'
  let size = `${spacing}px ${spacing}px`
  if (direction === 'vertical') {
    size = `${spacing * 0.7}px ${spacing}px`
  } else if (direction === 'diagonal') {
    // A tiled background can't literally rotate the tile axis with pure
    // background-position tricks, so "diagonal" staggers alternating rows
    // instead, which reads as a diagonal rhythm at a glance.
    position = `0 0, ${spacing / 2}px ${spacing / 2}px`
    size = `${spacing}px ${spacing}px`
  }

  return `${dataUri} ${position}/${size} repeat, ${baseColor}`
}

export interface ParsedPatternBackground {
  patternId: string
  baseColor: string
  motifColor: string
  direction: PatternDirection
  spacing: number
}

const PATTERN_BG_RE = /^url\("data:image\/svg\+xml,([^"]+)"\)\s+([^/]+)\/([^,]+),\s*(#[0-9a-fA-F]{3,8})$/

export function parsePatternBackground(value: string): ParsedPatternBackground | null {
  const match = PATTERN_BG_RE.exec(value.trim())
  if (!match) return null

  try {
    const decoded = decodeURIComponent(match[1])
    const colorMatch = /color="(#[0-9a-fA-F]{3,8})"/.exec(decoded)
    const motifMatch = PATTERNS.find((p) => decoded.includes(p.motif.slice(0, 20)))
    const sizeParts = match[3].trim().split(/\s+/)
    const spacing = parseInt(sizeParts[0], 10) || 40
    const position = match[2].trim()

    return {
      patternId: motifMatch?.id ?? PATTERNS[0].id,
      baseColor: match[4],
      motifColor: colorMatch?.[1] ?? '#000000',
      direction: position.includes(',') ? 'diagonal' : sizeParts.length > 1 && sizeParts[0] !== sizeParts[1] ? 'vertical' : 'horizontal',
      spacing,
    }
  } catch {
    return null
  }
}

export function isPatternBackground(value: string): boolean {
  return PATTERN_BG_RE.test(value.trim())
}
