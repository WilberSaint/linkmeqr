<script setup lang="ts">
import { computed } from 'vue'
import type { ProfileTheme } from '@/types'
import { AVAILABLE_FONTS, ensureGoogleFontLoaded } from '@/composables/useGoogleFont'
import { BACKGROUND_PRESETS } from '@/composables/backgroundPresets'
import { buildGradient, contrastingTextColor, isSimpleLinearGradient } from '@/composables/gradientUtils'
import { PATTERNS, buildPatternBackground, isPatternBackground } from '@/composables/backgroundPatterns'
import GradientEditor from './GradientEditor.vue'
import PatternEditor from './PatternEditor.vue'
import ColorInput from './ColorInput.vue'

const props = defineProps<{ theme: ProfileTheme; hasLogoImage: boolean; backgroundUploading?: boolean }>()
const emit = defineEmits<{ update: [payload: Partial<ProfileTheme>]; uploadBackground: [file: File] }>()

type BackgroundTab = 'color' | 'gradient' | 'pattern' | 'image'

// 'pattern' and 'image' are temporarily hidden from the tab bar while they
// get polished — if a profile already has one saved (from earlier testing),
// treat it like 'gradient'/'color' for tab purposes so the UI never lands on
// a tab with no button to select it; the underlying value is untouched.
const backgroundTab = computed<BackgroundTab>({
  get() {
    if (props.theme.background_type === 'gradient' || isSimpleLinearGradient(props.theme.background_value)) return 'gradient'
    return 'color'
  },
  set(tab) {
    if (tab === 'color') {
      emit('update', { background_type: 'color', background_value: props.theme.background_type === 'color' ? props.theme.background_value : '#ffffff' })
    } else if (tab === 'gradient') {
      emit('update', { background_type: 'gradient', background_value: buildGradient(160, '#6366f1', '#a78bfa') })
    } else if (tab === 'pattern') {
      emit('update', { background_type: 'pattern', background_value: buildPatternBackground(PATTERNS[0].id, '#ffffff', '#111827', 'horizontal') })
    } else {
      emit('update', { background_type: 'image', background_value: '' })
    }
  },
})

function onBackgroundFileSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  ;(e.target as HTMLInputElement).value = ''
  if (file) emit('uploadBackground', file)
}

const buttonStyles: { value: ProfileTheme['button_style']; label: string }[] = [
  { value: 'rounded', label: 'Redondeado' },
  { value: 'square', label: 'Cuadrado' },
  { value: 'pill', label: 'Píldora' },
  { value: 'outline', label: 'Contorno' },
]

const logoShapes: { value: ProfileTheme['logo_shape']; label: string; class: string }[] = [
  { value: 'circle', label: 'Círculo', class: 'rounded-full' },
  { value: 'rounded', label: 'Redondeado', class: 'rounded-md' },
  { value: 'square', label: 'Cuadrado', class: 'rounded-none' },
]

const colorPresets = computed(() => BACKGROUND_PRESETS.filter((p) => p.type === 'color'))
const gradientPresets = computed(() => BACKGROUND_PRESETS.filter((p) => p.type === 'gradient' && isSimpleLinearGradient(p.value)))

const activePresetId = computed(() => {
  const match = BACKGROUND_PRESETS.find((p) => p.value === props.theme.background_value)
  return match?.id ?? null
})

function applyPreset(presetId: string) {
  const preset = BACKGROUND_PRESETS.find((p) => p.id === presetId)
  if (!preset) return
  emit('update', { background_type: preset.type, background_value: preset.value })
}

function onSolidColor(value: string) {
  emit('update', { background_type: 'color', background_value: value })
}

function onFontChange(e: Event) {
  const family = (e.target as HTMLSelectElement).value
  ensureGoogleFontLoaded(family)
  emit('update', { font_family: family })
}

// Shortcut that syncs the logo circle, button background, and button text
// (with automatic contrast) to the same color in one click — lives after the
// individual color rows so its effect is obvious once you've seen each one.
function setAccentColor(color: string) {
  emit('update', {
    secondary_color: color,
    logo_background_color: color,
    button_text_color: contrastingTextColor(color),
  })
}

for (const f of AVAILABLE_FONTS) ensureGoogleFontLoaded(f)
</script>

<template>
  <div class="space-y-6">
    <section>
      <h3 class="text-[11px] font-semibold text-gray-400 uppercase tracking-wide mb-2.5">Fondo de la página</h3>

      <div class="flex gap-1 mb-3 bg-gray-100 rounded-lg p-1">
        <button
          v-for="t in [
            { id: 'color', label: 'Sólido' },
            { id: 'gradient', label: 'Degradado' },
          ]"
          :key="t.id"
          type="button"
          class="flex-1 rounded-md py-1.5 text-[11px] font-medium transition"
          :class="backgroundTab === t.id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'"
          @click="backgroundTab = t.id as BackgroundTab"
        >
          {{ t.label }}
        </button>
      </div>

      <div v-if="backgroundTab === 'color'">
        <div class="grid grid-cols-5 gap-2 mb-3">
          <button
            v-for="preset in colorPresets"
            :key="preset.id"
            type="button"
            class="aspect-square rounded-lg border-2 transition"
            :class="activePresetId === preset.id ? 'border-indigo-500 ring-2 ring-indigo-200' : 'border-gray-200 hover:border-gray-300'"
            :style="{ background: preset.swatch }"
            :title="preset.label"
            @click="applyPreset(preset.id)"
          />
        </div>
        <ColorInput
          :model-value="theme.background_type === 'color' ? theme.background_value : '#ffffff'"
          @update:model-value="onSolidColor"
        />
      </div>

      <div v-else-if="backgroundTab === 'gradient'">
        <div class="grid grid-cols-5 gap-2 mb-3">
          <button
            v-for="preset in gradientPresets"
            :key="preset.id"
            type="button"
            class="aspect-square rounded-lg border-2 transition"
            :class="activePresetId === preset.id ? 'border-indigo-500 ring-2 ring-indigo-200' : 'border-gray-200 hover:border-gray-300'"
            :style="{ background: preset.swatch }"
            :title="preset.label"
            @click="applyPreset(preset.id)"
          />
        </div>
        <GradientEditor
          :value="isSimpleLinearGradient(theme.background_value) ? theme.background_value : buildGradient(160, '#6366f1', '#a78bfa')"
          @update="(value) => emit('update', { background_type: 'gradient', background_value: value })"
        />
      </div>

      <PatternEditor
        v-else-if="backgroundTab === 'pattern'"
        :value="isPatternBackground(theme.background_value) ? theme.background_value : buildPatternBackground(PATTERNS[0].id, '#ffffff', '#111827', 'horizontal')"
        @update="(value) => emit('update', { background_type: 'pattern', background_value: value })"
      />

      <div v-else-if="backgroundTab === 'image'">
        <div v-if="theme.background_url" class="mb-3 relative">
          <img :src="theme.background_url" alt="Fondo" class="w-full h-24 object-cover rounded-lg border border-gray-200" />
        </div>
        <label class="inline-flex items-center gap-2 text-xs font-medium text-indigo-600 hover:underline cursor-pointer">
          {{ backgroundUploading ? 'Subiendo…' : theme.background_url ? 'Cambiar imagen' : 'Subir imagen' }}
          <input
            type="file"
            accept="image/png,image/jpeg,image/gif,image/webp"
            class="hidden"
            :disabled="backgroundUploading"
            @change="onBackgroundFileSelected"
          />
        </label>
        <p class="text-[11px] text-gray-400 mt-1">Se ajustará para cubrir todo el fondo de tu página.</p>
      </div>
    </section>

    <section>
      <h3 class="text-[11px] font-semibold text-gray-400 uppercase tracking-wide mb-2.5">Logo del negocio</h3>
      <div class="flex gap-2 mb-3">
        <button
          type="button"
          class="flex-1 rounded-lg border-2 px-3 py-2 text-xs font-medium transition disabled:opacity-40 disabled:cursor-not-allowed"
          :class="theme.logo_display_mode === 'image'
            ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
            : 'border-gray-200 text-gray-500 hover:border-gray-300'"
          :disabled="!hasLogoImage"
          :title="!hasLogoImage ? 'Sube una imagen primero, en la pestaña Contenido' : ''"
          @click="emit('update', { logo_display_mode: 'image' })"
        >
          Usar imagen
        </button>
        <button
          type="button"
          class="flex-1 rounded-lg border-2 px-3 py-2 text-xs font-medium transition"
          :class="theme.logo_display_mode === 'initial'
            ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
            : 'border-gray-200 text-gray-500 hover:border-gray-300'"
          @click="emit('update', { logo_display_mode: 'initial' })"
        >
          Usar inicial (letra)
        </button>
      </div>

      <label class="block text-xs font-medium text-gray-600 mb-1.5">Forma</label>
      <div class="flex gap-2">
        <button
          v-for="shape in logoShapes"
          :key="shape.value"
          type="button"
          class="flex-1 flex flex-col items-center gap-1 rounded-lg border-2 py-2 transition"
          :class="theme.logo_shape === shape.value ? 'border-indigo-500 bg-indigo-50' : 'border-gray-200 hover:border-gray-300'"
          @click="emit('update', { logo_shape: shape.value })"
        >
          <span class="w-7 h-7 bg-gray-400" :class="shape.class" />
          <span class="text-[11px] text-gray-600">{{ shape.label }}</span>
        </button>
      </div>

      <div v-if="theme.logo_display_mode === 'initial'" class="flex items-center gap-3 mt-4">
        <div
          class="w-9 h-9 rounded-full shrink-0 flex items-center justify-center text-sm font-semibold ring-2 ring-offset-2 ring-gray-200"
          :style="{ backgroundColor: theme.logo_background_color, color: theme.logo_text_color }"
        >
          A
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-xs font-medium text-gray-700">Color de la letra</p>
          <p class="text-[11px] text-gray-400">Debe contrastar con el fondo del círculo</p>
        </div>
        <ColorInput
          :model-value="theme.logo_text_color"
          @update:model-value="(v) => emit('update', { logo_text_color: v })"
        />
      </div>
    </section>

    <section>
      <h3 class="text-[11px] font-semibold text-gray-400 uppercase tracking-wide mb-2.5">Colores</h3>

      <div class="space-y-3">
        <div class="flex items-center gap-3">
          <div
            class="w-9 h-9 rounded-full shrink-0 flex items-center justify-center text-white text-sm font-semibold ring-2 ring-offset-2 ring-gray-200"
            :style="{ backgroundColor: theme.logo_background_color }"
          >
            A
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-gray-700">Fondo del ícono del negocio</p>
            <p class="text-[11px] text-gray-400">El círculo con tu inicial, cuando no usas una foto</p>
          </div>
          <ColorInput
            :model-value="theme.logo_background_color"
            @update:model-value="(v) => emit('update', { logo_background_color: v })"
          />
        </div>

        <div class="flex items-center gap-3">
          <div
            class="w-9 h-9 rounded-lg shrink-0 flex items-center justify-center text-[9px] font-medium"
            :style="{ backgroundColor: theme.secondary_color, color: theme.button_text_color }"
          >
            Btn
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-gray-700">{{ theme.button_style === 'outline' ? 'Borde y texto de los botones' : 'Fondo de los botones' }}</p>
            <p class="text-[11px] text-gray-400">Instagram, WhatsApp, enlaces… todos tus bloques</p>
          </div>
          <ColorInput
            :model-value="theme.secondary_color"
            @update:model-value="(v) => emit('update', { secondary_color: v })"
          />
        </div>

        <div v-if="theme.button_style !== 'outline'" class="flex items-center gap-3">
          <div
            class="w-9 h-9 rounded-lg shrink-0 flex items-center justify-center text-[9px] font-medium border border-gray-200"
            :style="{ backgroundColor: theme.secondary_color, color: theme.button_text_color }"
          >
            Aa
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-gray-700">Letras dentro de los botones</p>
            <p class="text-[11px] text-gray-400">Debe contrastar con el fondo del botón de arriba</p>
          </div>
          <ColorInput
            :model-value="theme.button_text_color"
            @update:model-value="(v) => emit('update', { button_text_color: v })"
          />
        </div>

        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg shrink-0 flex flex-col items-center justify-center gap-0.5 border border-gray-200 bg-white overflow-hidden px-1">
            <span class="text-[9px] font-semibold leading-none truncate w-full text-center" :style="{ color: theme.text_color }">Mi negocio</span>
            <span class="text-[7px] leading-none truncate w-full text-center opacity-70" :style="{ color: theme.text_color }">Descripción aquí</span>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-gray-700">Nombre y descripción del negocio</p>
            <p class="text-[11px] text-gray-400">El texto debajo de tu logo, en la página pública</p>
          </div>
          <ColorInput
            :model-value="theme.text_color"
            @update:model-value="(v) => emit('update', { text_color: v })"
          />
        </div>
      </div>

      <div class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100">
        <span class="text-[11px] text-gray-500 shrink-0">Aplicar el mismo color a los 3 de arriba:</span>
        <button
          type="button"
          class="rounded-md border border-gray-300 bg-black text-white text-[11px] font-medium px-3 py-1"
          @click="setAccentColor('#000000')"
        >
          Negro
        </button>
        <button
          type="button"
          class="rounded-md border border-gray-300 bg-white text-gray-900 text-[11px] font-medium px-3 py-1"
          @click="setAccentColor('#ffffff')"
        >
          Blanco
        </button>
      </div>
    </section>

    <section>
      <h3 class="text-[11px] font-semibold text-gray-400 uppercase tracking-wide mb-2.5">Tipografía y botones</h3>
      <label class="block text-xs font-medium text-gray-600 mb-1">Tipografía</label>
      <select
        :value="theme.font_family"
        class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
        @change="onFontChange"
      >
        <option v-for="f in AVAILABLE_FONTS" :key="f" :value="f" :style="{ fontFamily: f }">{{ f }}</option>
      </select>
      <p class="mt-1.5 text-lg" :style="{ fontFamily: theme.font_family }">Vista previa: Aa Bb Cc</p>

      <label class="block text-xs font-medium text-gray-600 mb-1 mt-4">Estilo de botones</label>
      <select
        :value="theme.button_style"
        class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
        @change="emit('update', { button_style: ($event.target as HTMLSelectElement).value as ProfileTheme['button_style'] })"
      >
        <option v-for="s in buttonStyles" :key="s.value" :value="s.value">{{ s.label }}</option>
      </select>

      <label class="flex items-center gap-2 text-sm text-gray-700 mt-3">
        <input
          type="checkbox"
          :checked="theme.button_shadow"
          @change="emit('update', { button_shadow: ($event.target as HTMLInputElement).checked })"
        />
        Sombra en botones
      </label>

      <label class="block text-xs font-medium text-gray-600 mb-1.5 mt-4">Disposición de los bloques</label>
      <div class="flex gap-2">
        <button
          type="button"
          class="flex-1 rounded-lg border-2 px-3 py-2 text-xs font-medium transition"
          :class="theme.layout === 'grid' ? 'border-gray-200 text-gray-500 hover:border-gray-300' : 'border-indigo-500 bg-indigo-50 text-indigo-700'"
          @click="emit('update', { layout: 'list' })"
        >
          Lista
        </button>
        <button
          type="button"
          class="flex-1 rounded-lg border-2 px-3 py-2 text-xs font-medium transition"
          :class="theme.layout === 'grid' ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-500 hover:border-gray-300'"
          @click="emit('update', { layout: 'grid' })"
        >
          Cuadrícula
        </button>
      </div>
    </section>
  </div>
</template>
