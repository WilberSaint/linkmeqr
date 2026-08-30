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

/** Converts a #rrggbb hex color plus an opacity (0–1) into an rgba() string. */
export function hexToRgba(hex: string, opacity: number): string {
  const clean = hex.replace('#', '')
  const full = clean.length === 3 ? clean.split('').map((c) => c + c).join('') : clean
  const r = parseInt(full.slice(0, 2), 16) || 0
  const g = parseInt(full.slice(2, 4), 16) || 0
  const b = parseInt(full.slice(4, 6), 16) || 0
  const a = Math.min(1, Math.max(0, opacity))
  return `rgba(${r}, ${g}, ${b}, ${a})`
}

/** WCAG relative luminance (0=black, 1=white) of a #rrggbb hex color. */
function relativeLuminance(hex: string): number {
  const clean = hex.replace('#', '')
  if (clean.length !== 6) return 0
  const r = parseInt(clean.slice(0, 2), 16) / 255
  const g = parseInt(clean.slice(2, 4), 16) / 255
  const b = parseInt(clean.slice(4, 6), 16) / 255
  const toLinear = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
  return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
}

/** Picks black or white text for best contrast against a given hex background. */
export function contrastingTextColor(hex: string): '#000000' | '#ffffff' {
  return relativeLuminance(hex) > 0.5 ? '#000000' : '#ffffff'
}

/**
 * WCAG contrast ratio between two hex colors — 1 (identical) to 21 (black on
 * white). 4.5 is the usual "readable body text" threshold; below ~2 two
 * colors read as nearly the same, which is the case worth flagging here
 * (this is a card background vs. the page's own text color, not audited
 * body-text-on-white, so the bar is "visibly different", not full AA).
 */
export function contrastRatio(hexA: string, hexB: string): number {
  const lA = relativeLuminance(hexA)
  const lB = relativeLuminance(hexB)
  const lighter = Math.max(lA, lB)
  const darker = Math.min(lA, lB)
  return (lighter + 0.05) / (darker + 0.05)
}
