/**
 * Texture tile generators — mirrors dotPatternDef/linePatternDef/etc. in the
 * Go renderer (print_card_icons.go) field for field. The canvas never drew
 * any of these: the card's own background pattern picker updated the store
 * correctly, but nothing read `background.pattern` when painting the SVG, so
 * every texture looked broken from inside the editor even though the actual
 * export already rendered it fine. This is what closes that gap, and what
 * lets a shape element wear the same textures the card background can.
 */

export type PatternKind = '' | 'dots' | 'lines' | 'grid' | 'waves' | 'circles'

export const PATTERN_OPTIONS: { value: PatternKind; label: string }[] = [
  { value: '', label: 'Ninguna' },
  { value: 'dots', label: 'Puntos' },
  { value: 'lines', label: 'Líneas' },
  { value: 'grid', label: 'Cuadrícula' },
  { value: 'waves', label: 'Ondas' },
  { value: 'circles', label: 'Círculos' },
]

/** spacing/opacity per kind, as a fraction of the shortest side of whatever box the pattern fills — matches backgroundPatternDef's own constants exactly. */
const PATTERN_TUNING: Record<Exclude<PatternKind, ''>, { spacingFrac: number; opacity: number }> = {
  dots: { spacingFrac: 0.045, opacity: 0.14 },
  lines: { spacingFrac: 0.05, opacity: 0.12 },
  grid: { spacingFrac: 0.05, opacity: 0.16 },
  waves: { spacingFrac: 0.04, opacity: 0.16 },
  circles: { spacingFrac: 0.05, opacity: 0.16 },
}

function dotPatternDef(id: string, spacing: number, color: string, opacity: number): string {
  return `<pattern id="${id}" width="${spacing}" height="${spacing}" patternUnits="userSpaceOnUse"><circle cx="${spacing / 2}" cy="${spacing / 2}" r="${spacing * 0.12}" fill="${color}" opacity="${opacity}"/></pattern>`
}

function linePatternDef(id: string, spacing: number, color: string, opacity: number): string {
  return `<pattern id="${id}" width="${spacing}" height="${spacing}" patternUnits="userSpaceOnUse" patternTransform="rotate(45)"><line x1="0" y1="0" x2="0" y2="${spacing}" stroke="${color}" stroke-width="${spacing * 0.14}" opacity="${opacity}"/></pattern>`
}

function gridPatternDef(id: string, spacing: number, color: string, opacity: number): string {
  return `<pattern id="${id}" width="${spacing}" height="${spacing}" patternUnits="userSpaceOnUse"><path d="M ${spacing} 0 L 0 0 0 ${spacing}" stroke="${color}" stroke-width="${spacing * 0.06}" opacity="${opacity}" fill="none"/></pattern>`
}

function wavesPatternDef(id: string, spacing: number, color: string, opacity: number): string {
  const r = spacing / 2
  const path = `M 0 ${r} A ${r} ${r} 0 0 1 ${spacing} ${r} A ${r} ${r} 0 0 0 ${spacing * 2} ${r}`
  return `<pattern id="${id}" width="${spacing * 2}" height="${spacing}" patternUnits="userSpaceOnUse"><path d="${path}" stroke="${color}" stroke-width="${spacing * 0.1}" opacity="${opacity}" fill="none"/></pattern>`
}

function circlesPatternDef(id: string, spacing: number, color: string, opacity: number): string {
  return `<pattern id="${id}" width="${spacing}" height="${spacing}" patternUnits="userSpaceOnUse"><circle cx="${spacing / 2}" cy="${spacing / 2}" r="${spacing * 0.35}" stroke="${color}" stroke-width="${spacing * 0.06}" opacity="${opacity}" fill="none"/></pattern>`
}

/**
 * Builds the `<pattern>` def for one texture, sized off `base` (the shortest
 * side of the box it will fill — a card's own canvas, or one shape
 * element's own box). Returns '' for no pattern, matching
 * backgroundPatternDef's own "unknown/none → skip" fallback.
 */
export function patternDef(id: string, kind: PatternKind, base: number, color: string): string {
  if (!kind) return ''
  const tuning = PATTERN_TUNING[kind]
  const spacing = base * tuning.spacingFrac
  switch (kind) {
    case 'dots':
      return dotPatternDef(id, spacing, color, tuning.opacity)
    case 'lines':
      return linePatternDef(id, spacing, color, tuning.opacity)
    case 'grid':
      return gridPatternDef(id, spacing, color, tuning.opacity)
    case 'waves':
      return wavesPatternDef(id, spacing, color, tuning.opacity)
    case 'circles':
      return circlesPatternDef(id, spacing, color, tuning.opacity)
  }
}
