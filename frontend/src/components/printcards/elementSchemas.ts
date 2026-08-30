import type { ElementProps, ElementType } from '@/types/cardLayout'
import { resolveCaptionState } from './captionSegments'
import { PATTERN_OPTIONS } from './patterns'

/**
 * What each element type exposes as editable properties, described as data.
 *
 * This is the piece that replaces the old per-template form. The properties
 * panel renders whatever this file declares for the selected element's type,
 * so adding a property to shapes means adding one line here — not a new
 * branch in a form that already knew what a "thank-you card" was.
 */

// 'phrase' is 'select' plus an escape hatch: a curated list of quick picks,
// with a "Personalizado…" option that reveals a free-text input for anything
// else. Used wherever the field has good defaults worth offering but must
// never refuse an arbitrary value.
export type FieldKind = 'text' | 'multiline' | 'number' | 'color' | 'select' | 'toggle' | 'range' | 'phrase'

export interface PropertyField {
  /** Key within the element's props bag. */
  key: keyof ElementProps
  label: string
  kind: FieldKind
  group: string
  options?: { value: string | number; label: string }[]
  min?: number
  max?: number
  step?: number
  /** Hides the field unless the element's current props satisfy this. */
  showIf?: (props: ElementProps) => boolean
  hint?: string
}

const FONT_FAMILIES = [
  { value: 'Arial, sans-serif', label: 'Arial' },
  { value: 'Georgia, serif', label: 'Georgia' },
  { value: '"Times New Roman", serif', label: 'Times' },
  { value: '"Courier New", monospace', label: 'Courier' },
  { value: 'Verdana, sans-serif', label: 'Verdana' },
  { value: '"Trebuchet MS", sans-serif', label: 'Trebuchet' },
]

const FONT_WEIGHTS = [
  { value: 400, label: 'Normal' },
  { value: 600, label: 'Semibold' },
  { value: 700, label: 'Bold' },
  { value: 800, label: 'Extra bold' },
]

export const ICON_NAMES = [
  { value: 'google', label: 'Google' },
  { value: 'instagram', label: 'Instagram' },
  { value: 'facebook', label: 'Facebook' },
  { value: 'whatsapp', label: 'WhatsApp' },
  { value: 'youtube', label: 'YouTube' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'star', label: 'Estrella' },
  { value: 'heart', label: 'Corazón' },
  { value: 'gift', label: 'Regalo' },
  { value: 'menu', label: 'Menú' },
  { value: 'pin', label: 'Ubicación' },
  { value: 'contactless', label: 'Ondas NFC' },
  { value: 'tap_hand', label: 'Dedo tocando' },
  { value: 'tap_card', label: 'Tarjeta/teléfono' },
  { value: 'scan', label: 'Visor de cámara' },
  { value: 'scan_qr', label: 'Código QR' },
  { value: 'scan_camera', label: 'Cámara' },
]

/** Icon glyphs that read as "tap here" — offered for the Toca prompt of a QR's caption. */
export const TAP_ICON_NAMES = [
  { value: 'contactless', label: 'Ondas NFC' },
  { value: 'tap_hand', label: 'Dedo tocando' },
  { value: 'tap_card', label: 'Tarjeta/teléfono' },
]

/** Icon glyphs that read as "scan here" — offered for the Escanea prompt of a QR's caption. */
export const SCAN_ICON_NAMES = [
  { value: 'scan', label: 'Visor de cámara' },
  { value: 'scan_qr', label: 'Código QR' },
  { value: 'scan_camera', label: 'Cámara' },
]

/**
 * Curated wording for each prompt. A 'phrase' field (see PropertiesPanel)
 * shows these as quick picks and still lets the business type anything else —
 * the presets are a starting point, not a restriction.
 */
export const TAP_TEXT_PRESETS = ['Toca', 'Toca aquí', 'Toca para abrir', 'Acerca tu teléfono']
export const SCAN_TEXT_PRESETS = ['Escanea', 'Escanea aquí', 'Escanea el código', 'Escanéame', 'Apunta tu cámara']

const TEXT_FIELDS: PropertyField[] = [
  { key: 'text', label: 'Contenido', kind: 'multiline', group: 'Texto', hint: 'Usa saltos de línea para dividir en varias líneas.' },
  { key: 'font_family', label: 'Tipografía', kind: 'select', group: 'Tipografía', options: FONT_FAMILIES },
  { key: 'font_size', label: 'Tamaño', kind: 'number', group: 'Tipografía', min: 1, max: 400, step: 0.5 },
  { key: 'font_weight', label: 'Grosor', kind: 'select', group: 'Tipografía', options: FONT_WEIGHTS },
  { key: 'line_height', label: 'Interlineado', kind: 'number', group: 'Tipografía', min: 0.8, max: 3, step: 0.05 },
  { key: 'letter_spacing', label: 'Espaciado', kind: 'number', group: 'Tipografía', min: -5, max: 20, step: 0.5 },
  { key: 'color', label: 'Color', kind: 'color', group: 'Apariencia' },
  {
    key: 'align',
    label: 'Alineación',
    kind: 'select',
    group: 'Apariencia',
    options: [
      { value: 'left', label: 'Izquierda' },
      { value: 'center', label: 'Centro' },
      { value: 'right', label: 'Derecha' },
    ],
  },
  { key: 'italic', label: 'Cursiva', kind: 'toggle', group: 'Apariencia' },
  { key: 'underline', label: 'Subrayado', kind: 'toggle', group: 'Apariencia' },
  { key: 'uppercase', label: 'Mayúsculas', kind: 'toggle', group: 'Apariencia' },
]

// 'source' is deliberately not a schema field here — PropertiesPanel renders
// a dedicated upload control for images instead (thumbnail + file picker +
// a "usar el logo" shortcut), since picking an image is an action, not a
// dropdown choice, and it needs the client id to actually upload anything.
const IMAGE_FIELDS: PropertyField[] = [
  {
    key: 'fit',
    label: 'Ajuste',
    kind: 'select',
    group: 'Imagen',
    options: [
      { value: 'cover', label: 'Rellenar (recorta)' },
      { value: 'contain', label: 'Contener (deja margen)' },
    ],
  },
  {
    key: 'shape',
    label: 'Forma',
    kind: 'select',
    group: 'Imagen',
    options: [
      { value: '', label: 'Rectángulo' },
      { value: 'circle', label: 'Círculo' },
      { value: 'rounded', label: 'Redondeado' },
      { value: 'square', label: 'Cuadrado' },
    ],
  },
  {
    key: 'radius',
    label: 'Radio',
    kind: 'number',
    group: 'Imagen',
    min: 0,
    max: 200,
    step: 1,
    showIf: (p) => p.shape === 'rounded',
  },
]

const QR_FIELDS: PropertyField[] = [
  // The destination itself is edited through QrTargetSelect, which needs the
  // client's own list of available targets — the panel renders that field
  // specially rather than declaring it here.
  {
    key: 'frame',
    label: 'Marco',
    kind: 'select',
    group: 'Marco',
    options: [
      { value: 'none', label: 'Sin marco' },
      { value: 'rounded', label: 'Redondeado' },
      { value: 'square', label: 'Cuadrado' },
    ],
  },
  { key: 'frame_fill', label: 'Color del marco', kind: 'color', group: 'Marco', showIf: (p) => p.frame !== 'none' },
  {
    key: 'frame_radius',
    label: 'Redondeo',
    kind: 'range',
    group: 'Marco',
    min: 0,
    max: 0.5,
    step: 0.01,
    showIf: (p) => p.frame === 'rounded',
  },
  {
    key: 'frame_pad',
    label: 'Margen interior',
    kind: 'range',
    group: 'Marco',
    min: 0,
    max: 0.6,
    step: 0.01,
    hint: 'Espacio entre el código y el borde del marco.',
    showIf: (p) => p.frame !== 'none',
  },
  { key: 'shadow', label: 'Sombra', kind: 'toggle', group: 'Marco' },
  // Toca and Escanea are each their own independent toggle + icon + wording —
  // handled by hand in PropertiesPanel, the same way the QR's destination is,
  // because turning either on has to migrate an old-format element onto the
  // new fields (see updateCaption there). Only the styling that applies to
  // whichever of the two is showing lives here as ordinary direct-write props.
  {
    key: 'caption_bare',
    label: 'Sin cápsula',
    kind: 'toggle',
    group: 'Estilo de la leyenda',
    hint: 'Quita el fondo redondeado y deja sólo el texto y los íconos.',
    showIf: (p) => resolveCaptionState(p).showTap || resolveCaptionState(p).showScan,
  },
  {
    key: 'caption_size',
    label: 'Tamaño',
    kind: 'number',
    group: 'Estilo de la leyenda',
    min: 0,
    max: 60,
    step: 0.5,
    hint: '0 lo calcula a partir del tamaño del QR.',
    showIf: (p) => resolveCaptionState(p).showTap || resolveCaptionState(p).showScan,
  },
  {
    key: 'caption_fill',
    label: 'Color',
    kind: 'color',
    group: 'Estilo de la leyenda',
    showIf: (p) => resolveCaptionState(p).showTap || resolveCaptionState(p).showScan,
  },
]

const SHAPE_FIELDS: PropertyField[] = [
  {
    key: 'kind',
    label: 'Tipo',
    kind: 'select',
    group: 'Forma',
    options: [
      { value: 'rect', label: 'Rectángulo' },
      { value: 'ellipse', label: 'Elipse' },
      { value: 'line', label: 'Línea' },
      { value: 'polygon', label: 'Polígono' },
      { value: 'path', label: 'Trazo' },
    ],
  },
  { key: 'fill', label: 'Relleno', kind: 'color', group: 'Forma', showIf: (p) => p.kind !== 'line' },
  { key: 'radius', label: 'Esquinas', kind: 'number', group: 'Forma', min: 0, max: 200, step: 1, showIf: (p) => p.kind === 'rect' },
  { key: 'shadow', label: 'Sombra', kind: 'toggle', group: 'Forma' },
  { key: 'stroke', label: 'Borde', kind: 'color', group: 'Borde' },
  { key: 'stroke_width', label: 'Grosor', kind: 'number', group: 'Borde', min: 0, max: 40, step: 0.25 },
  { key: 'dash_array', label: 'Guiones', kind: 'text', group: 'Borde', hint: 'Ej. "4 3" para una línea discontinua.' },
  {
    key: 'pattern',
    label: 'Textura',
    kind: 'select',
    group: 'Textura',
    options: PATTERN_OPTIONS,
    showIf: (p) => p.kind !== 'line',
    hint: 'Se dibuja sobre el relleno, como en el fondo de la tarjeta.',
  },
  {
    key: 'pattern_ink',
    label: 'Color de la textura',
    kind: 'color',
    group: 'Textura',
    showIf: (p) => Boolean(p.pattern) && p.kind !== 'line',
  },
]

const ICON_FIELDS: PropertyField[] = [
  { key: 'name', label: 'Ícono', kind: 'select', group: 'Ícono', options: ICON_NAMES },
  { key: 'color', label: 'Color', kind: 'color', group: 'Ícono', hint: 'Los íconos de marca (Google, Instagram…) usan sus colores oficiales.' },
  { key: 'count', label: 'Repeticiones', kind: 'number', group: 'Ícono', min: 1, max: 10, step: 1 },
  { key: 'gap', label: 'Separación', kind: 'number', group: 'Ícono', min: 0.5, max: 4, step: 0.05, showIf: (p) => (p.count ?? 1) > 1 },
]

/** Every icon that reads as "tap" or "scan" — a prompt is not restricted to whichever one it started as. */
const PROMPT_ICON_NAMES = [{ value: '', label: 'Sin ícono' }, ...TAP_ICON_NAMES, ...SCAN_ICON_NAMES]

/** Both curated phrase sets combined — a prompt is free-standing, so it is no longer either "the Toca one" or "the Escanea one" specifically. */
const PROMPT_TEXT_PRESETS = [...TAP_TEXT_PRESETS, ...SCAN_TEXT_PRESETS]

const PROMPT_FIELDS: PropertyField[] = [
  { key: 'icon', label: 'Ícono', kind: 'select', group: 'Llamada a la acción', options: PROMPT_ICON_NAMES },
  {
    key: 'text',
    label: 'Texto',
    kind: 'phrase',
    group: 'Llamada a la acción',
    options: PROMPT_TEXT_PRESETS.map((phrase) => ({ value: phrase, label: phrase })),
  },
  { key: 'color', label: 'Color', kind: 'color', group: 'Llamada a la acción' },
  { key: 'bare', label: 'Sin cápsula', kind: 'toggle', group: 'Llamada a la acción', hint: 'Quita el fondo redondeado y deja sólo el ícono y el texto.' },
  {
    key: 'font_size',
    label: 'Tamaño',
    kind: 'number',
    group: 'Llamada a la acción',
    min: 0,
    max: 60,
    step: 0.5,
    hint: '0 lo calcula a partir del tamaño del elemento.',
  },
]

const SCHEMAS: Record<ElementType, PropertyField[]> = {
  text: TEXT_FIELDS,
  image: IMAGE_FIELDS,
  qr: QR_FIELDS,
  shape: SHAPE_FIELDS,
  icon: ICON_FIELDS,
  prompt: PROMPT_FIELDS,
}

export function schemaFor(type: ElementType): PropertyField[] {
  return SCHEMAS[type] ?? []
}

/** Fields grouped in declaration order, so the panel renders labelled sections. */
export function groupedSchema(type: ElementType, props: ElementProps): { group: string; fields: PropertyField[] }[] {
  const out: { group: string; fields: PropertyField[] }[] = []
  for (const field of schemaFor(type)) {
    if (field.showIf && !field.showIf(props)) continue
    let bucket = out.find((g) => g.group === field.group)
    if (!bucket) {
      bucket = { group: field.group, fields: [] }
      out.push(bucket)
    }
    bucket.fields.push(field)
  }
  return out
}
