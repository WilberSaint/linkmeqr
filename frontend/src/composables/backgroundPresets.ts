export interface BackgroundPreset {
  id: string
  label: string
  type: 'color' | 'gradient'
  value: string
  swatch: string // CSS background shorthand used to render the picker swatch
}

export const BACKGROUND_PRESETS: BackgroundPreset[] = [
  { id: 'white', label: 'Blanco', type: 'color', value: '#ffffff', swatch: '#ffffff' },
  { id: 'light-gray', label: 'Gris claro', type: 'color', value: '#f8fafc', swatch: '#f8fafc' },
  { id: 'cream', label: 'Crema', type: 'color', value: '#fefaf0', swatch: '#fefaf0' },
  { id: 'black', label: 'Negro', type: 'color', value: '#0b0f19', swatch: '#0b0f19' },
  { id: 'navy', label: 'Azul marino', type: 'color', value: '#0f172a', swatch: '#0f172a' },

  {
    id: 'sunset',
    label: 'Atardecer',
    type: 'gradient',
    value: 'linear-gradient(160deg,#f472b6,#facc15)',
    swatch: 'linear-gradient(160deg,#f472b6,#facc15)',
  },
  {
    id: 'ocean',
    label: 'Océano',
    type: 'gradient',
    value: 'linear-gradient(160deg,#0f172a,#4338ca)',
    swatch: 'linear-gradient(160deg,#0f172a,#4338ca)',
  },
  {
    id: 'forest',
    label: 'Bosque',
    type: 'gradient',
    value: 'linear-gradient(160deg,#064e3b,#16a34a)',
    swatch: 'linear-gradient(160deg,#064e3b,#16a34a)',
  },
  {
    id: 'fire',
    label: 'Fuego',
    type: 'gradient',
    value: 'linear-gradient(160deg,#7c2d12,#dc2626)',
    swatch: 'linear-gradient(160deg,#7c2d12,#dc2626)',
  },
  {
    id: 'lavender',
    label: 'Lavanda',
    type: 'gradient',
    value: 'linear-gradient(160deg,#6d28d9,#a78bfa)',
    swatch: 'linear-gradient(160deg,#6d28d9,#a78bfa)',
  },
  {
    id: 'candy',
    label: 'Dulce',
    type: 'gradient',
    value: 'linear-gradient(160deg,#db2777,#f97316)',
    swatch: 'linear-gradient(160deg,#db2777,#f97316)',
  },
  {
    id: 'mono',
    label: 'Grises',
    type: 'gradient',
    value: 'linear-gradient(160deg,#1f2937,#6b7280)',
    swatch: 'linear-gradient(160deg,#1f2937,#6b7280)',
  },

  // Subtle repeating patterns, expressed as CSS background shorthand (works
  // directly as background_value when background_type is 'gradient' since
  // both are plain CSS `background` values under the hood).
  {
    id: 'dots-light',
    label: 'Puntos claros',
    type: 'gradient',
    value: 'radial-gradient(circle,#cbd5e1 1.5px,transparent 1.5px) 0 0/16px 16px, #ffffff',
    swatch: 'radial-gradient(circle,#cbd5e1 1.5px,transparent 1.5px) 0 0/8px 8px, #ffffff',
  },
  {
    id: 'dots-dark',
    label: 'Puntos oscuros',
    type: 'gradient',
    value: 'radial-gradient(circle,#334155 1.5px,transparent 1.5px) 0 0/16px 16px, #0b0f19',
    swatch: 'radial-gradient(circle,#334155 1.5px,transparent 1.5px) 0 0/8px 8px, #0b0f19',
  },
  {
    id: 'diagonal-lines',
    label: 'Líneas diagonales',
    type: 'gradient',
    value: 'repeating-linear-gradient(45deg,#f1f5f9,#f1f5f9 10px,#ffffff 10px,#ffffff 20px)',
    swatch: 'repeating-linear-gradient(45deg,#f1f5f9,#f1f5f9 6px,#ffffff 6px,#ffffff 12px)',
  },
]
