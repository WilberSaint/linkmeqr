<script setup lang="ts">
/**
 * Draws one element of the card tree as SVG, inside the element's own
 * translated/rotated group. This is the browser-side twin of the Go
 * renderer's renderElement — same geometry conventions, same defaults — so
 * what the designer drags is what the exporter prints.
 *
 * Anything type-specific reads from `el.props`. Nothing here knows which
 * built-in design the card came from, which is the whole point of the tree.
 */
import { computed } from 'vue'

import type { CardElement } from '@/types/cardLayout'
import { iconCacheKey } from './assetCache'
import { captionRenderSegments } from './captionSegments'
import { patternDef, type PatternKind } from './patterns'

const props = defineProps<{
  el: CardElement
  /** Real QR artwork keyed by element id, fetched from the backend and cached. */
  qrSvgs: Record<string, string>
  /** Built-in glyph artwork keyed by iconCacheKey(name, color) — the exporter's own SVG. */
  iconSvgs: Record<string, string>
  /** Resolved image URLs keyed by element id (the business logo, or an upload). */
  imageSrcs: Record<string, string>
}>()

const el = computed(() => props.el)
const p = computed(() => props.el.props)

const transform = computed(() => {
  const e = el.value
  let t = `translate(${e.x},${e.y})`
  if (e.rotation) t += ` rotate(${e.rotation},${e.w / 2},${e.h / 2})`
  return t
})

const opacity = computed(() => (el.value.opacity === undefined ? 1 : el.value.opacity))

// --- text -------------------------------------------------------------------

const textLines = computed(() => {
  const raw = p.value.text ?? ''
  return (p.value.uppercase ? raw.toUpperCase() : raw).split('\n')
})

const fontSize = computed(() => p.value.font_size || el.value.h * 0.6)
const lineGap = computed(() => fontSize.value * (p.value.line_height || 1.2))

const textAnchor = computed(() => {
  if (p.value.align === 'left') return 'start'
  if (p.value.align === 'right') return 'end'
  return 'middle'
})

const textX = computed(() => {
  if (p.value.align === 'left') return 0
  if (p.value.align === 'right') return el.value.w
  return el.value.w / 2
})

/**
 * The block of lines is centered vertically in the box, with the first
 * baseline offset by roughly the cap height — identical to the Go renderer,
 * because the seeded trees encode their boxes assuming exactly this.
 */
const firstBaseline = computed(() => {
  const blockH = lineGap.value * (textLines.value.length - 1)
  return (el.value.h - blockH) / 2 + fontSize.value * 0.34
})

// --- qr ---------------------------------------------------------------------

/** A QR is always square and centered: a stretched one would not scan. */
const qrSide = computed(() => Math.min(el.value.w, el.value.h))
const qrPad = computed(() => (p.value.frame_pad && p.value.frame_pad > 0 ? p.value.frame_pad : 0.18))
const qrSize = computed(() => qrSide.value / (1 + qrPad.value))
const qrCx = computed(() => el.value.w / 2)
const qrCy = computed(() => el.value.h / 2)

const qrFrameRadius = computed(() => {
  if (p.value.frame === 'square') return 0
  const r = p.value.frame_radius
  return qrSide.value * (r && r > 0 ? r : 0.16)
})

const qrArtwork = computed(() => props.qrSvgs[el.value.id] ?? '')

/**
 * The fetched QR arrives as a standalone <svg> with its own module-unit
 * viewBox. Re-anchoring it as a nested <svg> viewport is what the Go renderer
 * does too, so the code keeps its own internal coordinate system untouched.
 */
const qrInner = computed(() => {
  if (!qrArtwork.value) return ''
  return qrArtwork.value.replace(
    /<svg /,
    `<svg x="${qrCx.value - qrSize.value / 2}" y="${qrCy.value - qrSize.value / 2}" width="${qrSize.value}" height="${qrSize.value}" `,
  )
})

const captionSize = computed(() => p.value.caption_size || qrSide.value * 0.12)
const captionCy = computed(() => qrCy.value + qrSide.value / 2 + captionSize.value * 1.9)

/**
 * The Toca/Escanea segments to draw, from the shared resolver — the layout
 * math below (drawScanCaption's arithmetic) is still a second implementation
 * of the Go renderer's, but WHICH segments and WHICH wording now come from
 * one place, so the canvas and the export can no longer disagree about that
 * part. The "Vista real" panel is what catches any drift in the geometry.
 */
const captionSegments = computed(() => captionRenderSegments(p.value))

/** Matches estTextWidth in Go — a deliberately rough metric, but the same rough metric. */
function estTextWidth(text: string, fontSize: number): number {
  return [...text].length * fontSize * 0.56
}

const captionLayout = computed(() => {
  const segments = captionSegments.value
  if (segments.length === 0) return null

  const fontSize = captionSize.value
  const iconD = fontSize * 1.05
  const innerGap = fontSize * 0.3
  const dividerGap = fontSize * 0.5
  const padX = fontSize * 0.55
  const pillH = fontSize * 2.1
  const cy = captionCy.value

  const widths = segments.map((seg) => estTextWidth(seg.label, fontSize) + (seg.icon ? iconD + innerGap : 0))
  const contentW = widths.reduce((a, b) => a + b, 0) + dividerGap * 2 * (segments.length - 1)
  const pillW = contentW + padX * 2
  const pillX = qrCx.value - pillW / 2

  const parts: {
    seg: { icon: string; label: string }
    iconCx: number | null
    labelX: number
    dividerX: number | null
  }[] = []

  let x = pillX + padX
  segments.forEach((seg, i) => {
    let dividerX: number | null = null
    if (i > 0) {
      dividerX = x + dividerGap
      x = dividerX + dividerGap
    }
    let iconCx: number | null = null
    let labelX = x
    if (seg.icon) {
      iconCx = x + iconD / 2
      labelX = iconCx + iconD / 2 + innerGap
    }
    parts.push({ seg, iconCx, labelX, dividerX })
    x += widths[i]
  })

  return {
    fontSize,
    iconD,
    pillX,
    pillY: cy - pillH / 2,
    pillW,
    pillH,
    textY: cy + fontSize * 0.32,
    iconCy: cy - fontSize * 0.06,
    dividerTop: cy - pillH * 0.3,
    dividerBottom: cy + pillH * 0.3,
    strokeW: Math.max(fontSize * 0.045, 1),
    parts,
  }
})

/** Caption glyphs come from the same cached backend artwork the icon element uses. */
function captionGlyph(name: string, cx: number, cy: number, r: number): string {
  const art = props.iconSvgs[iconCacheKey(name, p.value.caption_fill || '#111827')]
  if (!art) return ''
  return art.replace(/<svg /, `<svg x="${cx - r}" y="${cy - r}" width="${r * 2}" height="${r * 2}" `)
}

// --- prompt -------------------------------------------------------------------
//
// One "[icon] label" call-to-action — "Toca", "Escanea", or any other short
// instruction — scaled to fill its own box the same way a text or icon
// element does. Mirrors renderPromptElement in the Go renderer field for
// field: same shrink-to-fit floor, same pill math, so what the designer
// resizes here is exactly what the export draws.
const promptFontSize = computed(() => {
  let size = p.value.font_size || el.value.h / 1.4
  const measure = (s: number) => {
    let w = estTextWidth(p.value.text ?? '', s)
    if (p.value.icon) w += s * 1.05 + s * 0.3
    return w
  }
  const contentW = measure(size)
  if (contentW > el.value.w && contentW > 0) {
    size = Math.max((size * el.value.w) / contentW, size * 0.6)
  }
  return size
})

const promptLayout = computed(() => {
  const fontSize = promptFontSize.value
  const iconD = fontSize * 1.05
  const innerGap = fontSize * 0.3
  let contentW = estTextWidth(p.value.text ?? '', fontSize)
  if (p.value.icon) contentW += iconD + innerGap
  const cy = el.value.h / 2
  const startX = (el.value.w - contentW) / 2

  let pill: { x: number; y: number; w: number; h: number; rx: number } | null = null
  if (!p.value.bare) {
    const padX = fontSize * 0.55
    const pillH = fontSize * 2.1
    const pillW = contentW + padX * 2
    pill = { x: (el.value.w - pillW) / 2, y: cy - pillH / 2, w: pillW, h: pillH, rx: pillH / 2 }
  }

  let x = startX
  let iconCx: number | null = null
  if (p.value.icon) {
    iconCx = x + iconD / 2
    x = iconCx + iconD / 2 + innerGap
  }

  return { fontSize, cy, pill, iconCx, iconR: iconD / 2, textX: x, textY: cy + fontSize * 0.32, strokeW: Math.max(fontSize * 0.045, 1) }
})

function promptGlyph(name: string, cx: number, cy: number, r: number): string {
  const art = props.iconSvgs[iconCacheKey(name, p.value.color || '#111827')]
  if (!art) return ''
  return art.replace(/<svg /, `<svg x="${cx - r}" y="${cy - r}" width="${r * 2}" height="${r * 2}" `)
}

// --- shape ------------------------------------------------------------------

const shapeFill = computed(() => p.value.fill || 'none')
const shapeStroke = computed(() => (p.value.stroke && p.value.stroke_width ? p.value.stroke : 'none'))

/**
 * A shape's own texture — mirrors the Go renderer's shapePatternID/
 * writePatternDef: the same five card-background textures, now available on
 * any shape, layered over its solid fill rather than replacing it.
 */
const shapePatternId = computed(() => `shape-pattern-${el.value.id}`)
const shapePatternMarkup = computed(() => {
  if (!p.value.pattern) return ''
  const ink = p.value.pattern_ink || p.value.fill || '#111827'
  return patternDef(shapePatternId.value, p.value.pattern as PatternKind, Math.min(el.value.w, el.value.h), ink)
})
const shapePatternFill = computed(() => `url(#${shapePatternId.value})`)

// --- icon -------------------------------------------------------------------

const iconCount = computed(() => Math.max(1, p.value.count ?? 1))

/**
 * Icon rows fit inside the element's width: count glyphs of diameter d spaced
 * gap*d apart span d*(1 + gap*(count-1)). Same arithmetic as the exporter, so
 * a five-star strip lands on the same centers in both.
 *
 * Each glyph is the backend's own artwork, re-anchored as a nested <svg>
 * viewport the same way the QR is — drawing these from a second, hand-written
 * set of paths would let the on-screen icon drift from the printed one.
 */
const iconGlyphs = computed(() => {
  const e = el.value
  const count = iconCount.value
  const art = props.iconSvgs[iconCacheKey(p.value.name || 'star', p.value.color || '#111827')]
  if (!art) return []

  const place = (cx: number, cy: number, r: number) =>
    art.replace(/<svg /, `<svg x="${cx - r}" y="${cy - r}" width="${r * 2}" height="${r * 2}" `)

  if (count === 1) {
    return [place(e.w / 2, e.h / 2, Math.min(e.w, e.h) / 2)]
  }
  const gap = p.value.gap && p.value.gap > 0 ? p.value.gap : 1.3
  const d = e.w / (1 + gap * (count - 1))
  const r = Math.min(d / 2, e.h / 2)
  const spacing = d * gap
  const startX = e.w / 2 - (spacing * (count - 1)) / 2
  return Array.from({ length: count }, (_, i) => place(startX + i * spacing, e.h / 2, r))
})

// --- image ------------------------------------------------------------------

const imageClipId = computed(() => `clip-${el.value.id}`)
const imageHref = computed(() => props.imageSrcs[el.value.id] ?? '')
</script>

<template>
  <g :transform="transform" :opacity="opacity">
    <!-- text -->
    <template v-if="el.type === 'text'">
      <text
        v-for="(line, i) in textLines"
        :key="i"
        :x="textX"
        :y="firstBaseline + i * lineGap"
        :font-family="p.font_family || 'Arial, sans-serif'"
        :font-size="fontSize"
        :font-weight="p.font_weight || 400"
        :fill="p.color || '#111827'"
        :text-anchor="textAnchor"
        :letter-spacing="p.letter_spacing || undefined"
        :font-style="p.italic ? 'italic' : undefined"
        :text-decoration="p.underline ? 'underline' : undefined"
      >
        {{ line }}
      </text>
    </template>

    <!-- qr -->
    <template v-else-if="el.type === 'qr'">
      <rect
        v-if="p.frame && p.frame !== 'none'"
        :x="qrCx - qrSide / 2"
        :y="qrCy - qrSide / 2"
        :width="qrSide"
        :height="qrSide"
        :rx="qrFrameRadius"
        :fill="p.frame_fill || '#ffffff'"
        :filter="p.shadow ? 'url(#cardShadow)' : undefined"
      />
      <!-- eslint-disable-next-line vue/no-v-html -- backend-generated QR markup -->
      <g v-if="qrInner" v-html="qrInner" />
      <!-- Until the real code arrives, a dashed box marks the exact area it
           will occupy, so sizing it is never guesswork. -->
      <rect
        v-else
        :x="qrCx - qrSize / 2"
        :y="qrCy - qrSize / 2"
        :width="qrSize"
        :height="qrSize"
        fill="#e5e7eb"
        stroke="#9ca3af"
        stroke-width="1"
        stroke-dasharray="4 3"
      />
      <g v-if="captionLayout">
        <rect
          v-if="!p.caption_bare"
          :x="captionLayout.pillX"
          :y="captionLayout.pillY"
          :width="captionLayout.pillW"
          :height="captionLayout.pillH"
          :rx="captionLayout.pillH / 2"
          :fill="p.caption_fill || '#111827'"
          opacity="0.16"
          :stroke="p.caption_fill || '#111827'"
          :stroke-width="captionLayout.strokeW"
          stroke-opacity="0.55"
        />
        <template v-for="(part, i) in captionLayout.parts" :key="i">
          <line
            v-if="part.dividerX !== null && !p.caption_bare"
            :x1="part.dividerX"
            :y1="captionLayout.dividerTop"
            :x2="part.dividerX"
            :y2="captionLayout.dividerBottom"
            :stroke="p.caption_fill || '#111827'"
            :stroke-width="captionLayout.strokeW"
            opacity="0.4"
          />
          <!-- eslint-disable-next-line vue/no-v-html -- backend-generated glyph markup -->
          <g
            v-if="part.iconCx !== null"
            v-html="captionGlyph(part.seg.icon, part.iconCx, captionLayout.iconCy, captionLayout.iconD / 2)"
          />
          <text
            :x="part.labelX"
            :y="captionLayout.textY"
            :font-size="captionLayout.fontSize"
            font-weight="700"
            :fill="p.caption_fill || '#111827'"
          >
            {{ part.seg.label }}
          </text>
        </template>
      </g>
    </template>

    <!-- prompt: one "[icon] label" call-to-action, independent of any QR -->
    <template v-else-if="el.type === 'prompt'">
      <rect
        v-if="promptLayout.pill"
        :x="promptLayout.pill.x"
        :y="promptLayout.pill.y"
        :width="promptLayout.pill.w"
        :height="promptLayout.pill.h"
        :rx="promptLayout.pill.rx"
        :fill="p.color || '#111827'"
        opacity="0.16"
        :stroke="p.color || '#111827'"
        :stroke-width="promptLayout.strokeW"
        stroke-opacity="0.55"
      />
      <!-- eslint-disable-next-line vue/no-v-html -- backend-generated glyph markup -->
      <g
        v-if="p.icon && promptLayout.iconCx !== null"
        v-html="promptGlyph(p.icon, promptLayout.iconCx, promptLayout.cy, promptLayout.iconR)"
      />
      <text
        v-if="p.text"
        :x="promptLayout.textX"
        :y="promptLayout.textY"
        :font-size="promptLayout.fontSize"
        font-weight="700"
        :fill="p.color || '#111827'"
      >
        {{ p.text }}
      </text>
    </template>

    <!-- shape -->
    <template v-else-if="el.type === 'shape'">
      <!-- eslint-disable-next-line vue/no-v-html -- generated pattern markup, not user input -->
      <defs v-if="shapePatternMarkup" v-html="shapePatternMarkup" />
      <ellipse
        v-if="p.kind === 'ellipse'"
        :cx="el.w / 2"
        :cy="el.h / 2"
        :rx="el.w / 2"
        :ry="el.h / 2"
        :fill="shapeFill"
        :stroke="shapeStroke"
        :stroke-width="p.stroke_width || undefined"
        :stroke-dasharray="p.dash_array || undefined"
        :filter="p.shadow ? 'url(#cardShadow)' : undefined"
      />
      <line
        v-else-if="p.kind === 'line'"
        x1="0"
        :y1="el.h / 2"
        :x2="el.w"
        :y2="el.h / 2"
        :stroke="shapeStroke"
        :stroke-width="p.stroke_width || undefined"
        :stroke-dasharray="p.dash_array || undefined"
      />
      <polygon
        v-else-if="p.kind === 'polygon'"
        :points="p.points"
        :fill="shapeFill"
        :stroke="shapeStroke"
        :stroke-width="p.stroke_width || undefined"
      />
      <path
        v-else-if="p.kind === 'path'"
        :d="p.path"
        :fill="shapeFill"
        :stroke="shapeStroke"
        :stroke-width="p.stroke_width || undefined"
      />
      <rect
        v-else
        :width="el.w"
        :height="el.h"
        :rx="p.radius || 0"
        :fill="shapeFill"
        :stroke="shapeStroke"
        :stroke-width="p.stroke_width || undefined"
        :stroke-dasharray="p.dash_array || undefined"
        :filter="p.shadow ? 'url(#cardShadow)' : undefined"
      />

      <!-- The texture redraws the SAME geometry on top, filled with the
           pattern instead of the solid colour — exactly how the card's own
           background layers a texture over its fill, and why this needs no
           separate clip: the shape outline IS the boundary. -->
      <template v-if="p.pattern">
        <ellipse
          v-if="p.kind === 'ellipse'"
          :cx="el.w / 2"
          :cy="el.h / 2"
          :rx="el.w / 2"
          :ry="el.h / 2"
          :fill="shapePatternFill"
        />
        <polygon v-else-if="p.kind === 'polygon'" :points="p.points" :fill="shapePatternFill" />
        <path v-else-if="p.kind === 'path'" :d="p.path" :fill="shapePatternFill" />
        <rect v-else-if="p.kind !== 'line'" :width="el.w" :height="el.h" :rx="p.radius || 0" :fill="shapePatternFill" />
      </template>
    </template>

    <!-- icon -->
    <template v-else-if="el.type === 'icon'">
      <!-- eslint-disable-next-line vue/no-v-html -- backend-generated glyph markup -->
      <g v-for="(glyph, i) in iconGlyphs" :key="i" v-html="glyph" />
      <rect
        v-if="iconGlyphs.length === 0"
        :width="el.w"
        :height="el.h"
        fill="#f3f4f6"
        stroke="#d1d5db"
        stroke-width="1"
        stroke-dasharray="4 3"
      />
    </template>

    <!-- image -->
    <template v-else-if="el.type === 'image'">
      <defs>
        <clipPath :id="imageClipId">
          <ellipse
            v-if="p.shape === 'circle'"
            :cx="el.w / 2"
            :cy="el.h / 2"
            :rx="el.w / 2"
            :ry="el.h / 2"
          />
          <rect
            v-else
            :width="el.w"
            :height="el.h"
            :rx="p.shape === 'rounded' ? p.radius || Math.min(el.w, el.h) * 0.14 : 0"
          />
        </clipPath>
      </defs>
      <image
        v-if="imageHref"
        :width="el.w"
        :height="el.h"
        :href="imageHref"
        :preserveAspectRatio="p.fit === 'contain' ? 'xMidYMid meet' : 'xMidYMid slice'"
        :clip-path="`url(#${imageClipId})`"
      />
      <rect
        v-else
        :width="el.w"
        :height="el.h"
        fill="#f3f4f6"
        stroke="#d1d5db"
        stroke-width="1"
        stroke-dasharray="4 3"
        :clip-path="`url(#${imageClipId})`"
      />
    </template>
  </g>
</template>
