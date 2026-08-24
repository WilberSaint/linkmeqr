export interface ParsedGradient {
  angle: number
  colorA: string
  colorB: string
}

const GRADIENT_RE = /^linear-gradient\((\d+)deg\s*,\s*(#[0-9a-fA-F]{3,8})\s*,\s*(#[0-9a-fA-F]{3,8})\)$/

/** Parses a `linear-gradient(Ndeg,#hex,#hex)` string produced by buildGradient(). */
export function parseGradient(value: string): ParsedGradient | null {
  const match = GRADIENT_RE.exec(value.trim())
  if (!match) return null
  return { angle: Number(match[1]), colorA: match[2], colorB: match[3] }
}

export function buildGradient(angle: number, colorA: string, colorB: string): string {
  return `linear-gradient(${angle}deg,${colorA},${colorB})`
}

export function isSimpleLinearGradient(value: string): boolean {
  return GRADIENT_RE.test(value.trim())
}

/** Picks black or white text for best contrast against a given hex background. */
export function contrastingTextColor(hex: string): '#000000' | '#ffffff' {
  const clean = hex.replace('#', '')
  if (clean.length !== 6) return '#000000'
  const r = parseInt(clean.slice(0, 2), 16) / 255
  const g = parseInt(clean.slice(2, 4), 16) / 255
  const b = parseInt(clean.slice(4, 6), 16) / 255
  const toLinear = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
  const luminance = 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
  return luminance > 0.5 ? '#000000' : '#ffffff'
}
