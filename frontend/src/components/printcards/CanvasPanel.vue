<script setup lang="ts">
/**
 * Properties of the card itself rather than of any one element: its
 * background, its corner radius, and the business logo the design can draw
 * on.
 *
 * The background is not an element on purpose — nothing should be able to drag
 * the card's own backdrop off the card — which is why it needs a home of its
 * own instead of appearing in the element properties panel.
 */
import { computed, ref } from 'vue'

import * as adminQrApi from '@/api/adminQr'
import * as clientsApi from '@/api/clients'
import * as printCardsApi from '@/api/printCards'
import type { PrintCard } from '@/api/printCards'
import { useCardEditorStore } from '@/stores/cardEditor'
import ColorInput from '@/components/editor/ColorInput.vue'
import ImageCropModal from '@/components/editor/ImageCropModal.vue'
import { PATTERN_OPTIONS } from './patterns'

const props = defineProps<{ clientId: string; card: PrintCard; logoUrl?: string }>()
const emit = defineEmits<{ logoChanged: [url: string] }>()

const store = useCardEditorStore()

const background = computed(() => store.layout?.background ?? { fill: '#ffffff' })
const canvas = computed(() => store.layout?.canvas)

/** A gradient exists exactly when a second stop is set, so the toggle writes/clears that stop. */
const hasGradient = computed(() => Boolean(background.value.gradient_to))

const PATTERNS = PATTERN_OPTIONS

/** Quick starting points so a card can be recolored without picking every value by hand. */
const PRESETS: { label: string; fill: string; gradientTo?: string; ink: string }[] = [
  { label: 'Blanco', fill: '#ffffff', ink: '#111827' },
  { label: 'Crema', fill: '#fffaf2', ink: '#1f2430' },
  { label: 'Negro', fill: '#14141a', ink: '#ffffff' },
  { label: 'Índigo', fill: '#4f46e5', ink: '#ffffff' },
  { label: 'Esmeralda', fill: '#047857', ink: '#ffffff' },
  { label: 'Vino', fill: '#7f1d1d', ink: '#ffffff' },
  { label: 'Azul noche', fill: '#0f172a', gradientTo: '#4338ca', ink: '#ffffff' },
  { label: 'Atardecer', fill: '#b45309', gradientTo: '#be123c', ink: '#ffffff' },
]

function setFill(fill: string) {
  store.setBackground({ fill })
}

function setGradientTo(gradient_to: string) {
  store.setBackground({ gradient_to })
}

function toggleGradient(on: boolean) {
  // Turning a gradient off has to clear the second stop rather than blank it,
  // since the renderer treats "has a second stop" as "is a gradient".
  store.setBackground({ gradient_to: on ? background.value.gradient_to || '#111827' : '' })
}

function setPattern(pattern: string) {
  store.setBackground({ pattern: pattern as never })
}

function setPatternInk(pattern_ink: string) {
  store.setBackground({ pattern_ink })
}

function applyPreset(preset: (typeof PRESETS)[number]) {
  store.setBackground({
    fill: preset.fill,
    gradient_to: preset.gradientTo ?? '',
    pattern_ink: preset.ink,
  })
}

function setCornerRadius(raw: string) {
  const parsed = Number(raw)
  if (Number.isNaN(parsed) || !store.layout) return
  store.setCanvas({ corner_r: Math.max(0, parsed) })
}

// --- reset to the template's own colour --------------------------------

const restoringTemplate = ref(false)
const restoreError = ref('')

/**
 * Once you start tweaking the background there was no way back to what the
 * chosen design (template + style) actually shipped with — the seeded
 * colour only ever existed as the layout's starting point, nowhere else to
 * read it back from. This regenerates that same starting tree via the seed
 * endpoint (the card's own layout_key/style/content — its pre-tree fields,
 * still kept around for exactly this) and takes ONLY its background, so
 * every element the designer has since added, moved or restyled is left
 * completely alone — this restores the colour, not the whole design.
 */
async function restoreTemplateColor() {
  if (!confirm('¿Restaurar el color y la textura originales de la plantilla? No toca ningún elemento.')) return
  restoringTemplate.value = true
  restoreError.value = ''
  try {
    const res = await printCardsApi.seedCardLayout(props.clientId, {
      layout_key: props.card.layout_key,
      title: props.card.title,
      size_preset: props.card.size_preset,
      custom_width_cm: props.card.custom_width_cm,
      custom_height_cm: props.card.custom_height_cm,
      qr_target_type: props.card.qr_target_type,
      qr_target_value: props.card.qr_target_value,
      color_overrides: props.card.color_overrides,
      content: props.card.content,
    })
    store.setBackground(res.layout.background)
  } catch {
    restoreError.value = 'No se pudo restaurar el color de la plantilla.'
  } finally {
    restoringTemplate.value = false
  }
}

// --- business logo ----------------------------------------------------------

const logoUploading = ref(false)
const cropFile = ref<File | null>(null)
const cropOpen = ref(false)

function onLogoFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  cropFile.value = file
  cropOpen.value = true
}

async function onLogoCropped(blob: Blob, shape: 'circle' | 'rounded' | 'square') {
  cropOpen.value = false
  logoUploading.value = true
  try {
    const file = new File([blob], 'logo.png', { type: 'image/png' })
    const media = await adminQrApi.uploadClientMedia(props.clientId, file)
    await clientsApi.updateClientLogo(props.clientId, media.id, shape)
    emit('logoChanged', media.file_path)
  } finally {
    logoUploading.value = false
    cropFile.value = null
  }
}
</script>

<template>
  <div class="space-y-5 p-4">
    <!-- Background presets -->
    <section>
      <div class="mb-2 flex items-center justify-between">
        <h4 class="text-xs font-semibold uppercase tracking-wide text-gray-400">Fondo</h4>
        <button
          type="button"
          class="text-xs text-indigo-600 hover:underline disabled:text-gray-300"
          :disabled="restoringTemplate"
          title="Vuelve el color, el degradado y la textura a como los dejó la plantilla — no toca ningún elemento"
          @click="restoreTemplateColor"
        >
          {{ restoringTemplate ? 'Restaurando…' : 'Restaurar de la plantilla' }}
        </button>
      </div>
      <p v-if="restoreError" class="mb-2 text-xs text-red-600">{{ restoreError }}</p>
      <div class="mb-3 grid grid-cols-4 gap-2">
        <button
          v-for="preset in PRESETS"
          :key="preset.label"
          type="button"
          class="h-11 rounded-lg border border-gray-200 transition hover:scale-105 hover:border-indigo-400"
          :title="preset.label"
          :style="{
            background: preset.gradientTo
              ? `linear-gradient(135deg, ${preset.fill}, ${preset.gradientTo})`
              : preset.fill,
          }"
          @click="applyPreset(preset)"
        />
      </div>

      <label class="mb-2 block">
        <span class="mb-1 block text-xs text-gray-500">Color</span>
        <ColorInput :model-value="background.fill || '#ffffff'" @update:model-value="setFill" />
      </label>

      <label class="mb-2 flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          class="h-4 w-4"
          :checked="hasGradient"
          @change="toggleGradient(($event.target as HTMLInputElement).checked)"
        />
        <span>Degradado</span>
      </label>

      <label v-if="hasGradient" class="mb-2 block">
        <span class="mb-1 block text-xs text-gray-500">Segundo color</span>
        <ColorInput :model-value="background.gradient_to || '#111827'" @update:model-value="setGradientTo" />
      </label>
    </section>

    <!-- Texture -->
    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Textura</h4>
      <select
        class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
        :value="background.pattern ?? ''"
        @change="setPattern(($event.target as HTMLSelectElement).value)"
      >
        <option v-for="opt in PATTERNS" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
      <label v-if="background.pattern" class="mt-2 block">
        <span class="mb-1 block text-xs text-gray-500">Color de la textura</span>
        <ColorInput :model-value="background.pattern_ink || '#111827'" @update:model-value="setPatternInk" />
      </label>
    </section>

    <!-- Shape -->
    <section v-if="canvas">
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Forma</h4>
      <label class="block">
        <span class="mb-1 block text-xs text-gray-500">Esquinas redondeadas</span>
        <input
          type="range"
          min="0"
          :max="Math.min(canvas.w, canvas.h) / 2"
          step="1"
          class="w-full"
          :value="canvas.corner_r"
          @input="setCornerRadius(($event.target as HTMLInputElement).value)"
        />
      </label>
      <p class="text-xs text-gray-400">
        Tamaño: {{ (canvas.w / 100).toFixed(2) }} × {{ (canvas.h / 100).toFixed(2) }} pulgadas
      </p>
    </section>

    <!-- Business logo -->
    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Logo del negocio</h4>
      <div class="flex items-center gap-3">
        <label class="relative shrink-0 cursor-pointer">
          <img
            v-if="props.logoUrl"
            :src="props.logoUrl"
            alt="Logo"
            class="h-14 w-14 rounded-lg border border-gray-200 object-cover"
          />
          <div
            v-else
            class="flex h-14 w-14 items-center justify-center rounded-lg border-2 border-dashed border-gray-300 text-center text-[10px] text-gray-400"
          >
            {{ logoUploading ? '…' : 'Subir' }}
          </div>
          <input type="file" accept="image/*" class="hidden" :disabled="logoUploading" @change="onLogoFileSelected" />
        </label>
        <p class="flex-1 text-xs text-gray-500">
          Lo usan los elementos de imagen con origen «Logo del negocio», y el QR inmersivo.
        </p>
      </div>
    </section>

    <ImageCropModal
      :open="cropOpen"
      :file="cropFile"
      shape="circle"
      @close="cropOpen = false"
      @cropped="onLogoCropped"
    />
  </div>
</template>
