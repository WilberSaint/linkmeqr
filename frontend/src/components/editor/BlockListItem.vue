<script setup lang="ts">
import { GripVertical, Eye, EyeOff, Copy, Trash2, ChevronDown, ChevronUp, FileText } from '@lucide/vue'
import { computed, ref } from 'vue'
import type { ProfileBlock } from '@/types'
import { blockLabel } from '@/composables/blockLabels'
import ColorInput from './ColorInput.vue'
import GalleryFields from './blocks/GalleryFields.vue'
import HoursFields from './blocks/HoursFields.vue'
import TestimonialsFields from './blocks/TestimonialsFields.vue'
import MapFields from './blocks/MapFields.vue'

const props = defineProps<{ block: ProfileBlock; menuFileUploading?: boolean }>()
const emit = defineEmits<{
  /**
   * debounced tells the parent this came from a keystroke, so it can show the
   * change in the preview at once but coalesce the saves. Structural edits
   * leave it off and persist immediately.
   */
  update: [payload: Partial<ProfileBlock>, debounced?: boolean]
  remove: []
  duplicate: []
  uploadMenuFile: [file: File]
}>()

/** Text fields report on every keystroke so the live preview actually moves. */
function onType(payload: Partial<ProfileBlock>) {
  emit('update', payload, true)
}

const expanded = ref(false)

function onMenuFileSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  ;(e.target as HTMLInputElement).value = ''
  if (file) emit('uploadMenuFile', file)
}

const BRAND_COLORS: Record<string, string> = {
  instagram: '#E1306C',
  facebook: '#1877F2',
  whatsapp: '#25D366',
  tiktok: '#010101',
  youtube: '#FF0000',
}

const isBrandBlock = computed(() => props.block.block_type in BRAND_COLORS)

const styleOverride = computed<{ icon_color?: string; use_brand_color?: boolean }>(
  () => (props.block.style_overrides as { icon_color?: string; use_brand_color?: boolean } | null) ?? {},
)

// Three mutually exclusive color modes for a brand block's button:
// 'theme' (default, matches every other button), 'brand' (Instagram pink, etc.),
// or 'custom' (a specific color picked for just this block).
const colorMode = computed<'theme' | 'brand' | 'custom'>(() => {
  if (styleOverride.value.icon_color) return 'custom'
  if (styleOverride.value.use_brand_color === true) return 'brand'
  return 'theme'
})

const customColor = computed(() => styleOverride.value.icon_color ?? BRAND_COLORS[props.block.block_type] ?? '#6366f1')

function toggleVisible() {
  emit('update', { is_visible: !props.block.is_visible })
}

function setColorMode(mode: 'theme' | 'brand' | 'custom') {
  if (mode === 'theme') {
    emit('update', { style_overrides: null })
  } else if (mode === 'brand') {
    emit('update', { style_overrides: { use_brand_color: true } })
  } else {
    emit('update', { style_overrides: { icon_color: customColor.value } })
  }
}

function setCustomColor(color: string) {
  emit('update', { style_overrides: { icon_color: color } })
}
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-white" :class="block.is_visible ? '' : 'opacity-60'">
    <div class="flex items-center gap-2 px-3 py-2.5">
      <GripVertical :size="16" class="text-gray-300 drag-handle cursor-grab shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-gray-900 truncate">{{ block.title || blockLabel(block.block_type) }}</p>
        <p class="text-xs text-gray-400">{{ blockLabel(block.block_type) }}</p>
      </div>
      <button
        class="p-1.5 text-gray-400 hover:text-gray-700"
        :title="block.is_visible ? 'Ocultar del perfil' : 'Mostrar en el perfil'"
        :aria-label="block.is_visible ? 'Ocultar del perfil' : 'Mostrar en el perfil'"
        @click="toggleVisible"
      >
        <Eye v-if="block.is_visible" :size="16" />
        <EyeOff v-else :size="16" />
      </button>
      <button class="p-1.5 text-gray-400 hover:text-gray-700" title="Duplicar" aria-label="Duplicar" @click="emit('duplicate')">
        <Copy :size="16" />
      </button>
      <button class="p-1.5 text-gray-400 hover:text-red-600" title="Eliminar" aria-label="Eliminar" @click="emit('remove')">
        <Trash2 :size="16" />
      </button>
      <button
        class="p-1.5 text-gray-400 hover:text-gray-700"
        :title="expanded ? 'Contraer' : 'Editar'"
        :aria-label="expanded ? 'Contraer' : 'Editar'"
        @click="expanded = !expanded"
      >
        <ChevronUp v-if="expanded" :size="16" />
        <ChevronDown v-else :size="16" />
      </button>
    </div>

    <div v-if="expanded" class="px-3 pb-3 space-y-2 border-t border-gray-100 pt-2.5">
      <div>
        <label class="block text-xs font-medium text-gray-600 mb-1">Título</label>
        <input
          :value="block.title ?? ''"
          class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          @input="onType({ title: ($event.target as HTMLInputElement).value })"
        />
      </div>
      <div v-if="block.block_type === 'menu'">
        <label class="block text-xs font-medium text-gray-600 mb-1">Archivo del menú (PDF)</label>
        <div v-if="block.media_id" class="flex items-center gap-2 mb-2 px-2.5 py-2 rounded-lg border border-gray-200 bg-gray-50">
          <FileText :size="16" class="text-gray-400 shrink-0" />
          <span class="text-xs text-gray-600 truncate flex-1">Menú subido</span>
        </div>
        <label class="inline-flex items-center gap-2 text-xs font-medium text-indigo-600 hover:underline cursor-pointer">
          {{ menuFileUploading ? 'Subiendo…' : block.media_id ? 'Reemplazar archivo' : 'Subir PDF' }}
          <input type="file" accept="application/pdf" class="hidden" :disabled="menuFileUploading" @change="onMenuFileSelected" />
        </label>
        <p class="text-[11px] text-gray-400 mt-1">O usa un enlace externo en vez de subir un archivo:</p>
        <input
          :value="block.url ?? ''"
          placeholder="https://…"
          class="mt-1 w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          @input="onType({ url: ($event.target as HTMLInputElement).value })"
        />
      </div>
      <GalleryFields
        v-else-if="block.block_type === 'gallery'"
        :content="block.content"
        @update="(content) => emit('update', { content })"
      />
      <HoursFields
        v-else-if="block.block_type === 'hours'"
        :content="block.content"
        @update="(content) => emit('update', { content })"
      />
      <TestimonialsFields
        v-else-if="block.block_type === 'testimonials'"
        :content="block.content"
        @update="(content) => emit('update', { content })"
      />
      <MapFields
        v-else-if="block.block_type === 'map'"
        :content="block.content"
        @update="(content) => emit('update', { content })"
      />
      <div v-else-if="block.block_type !== 'text'">
        <label class="block text-xs font-medium text-gray-600 mb-1">
          {{ block.block_type === 'google_review' ? 'Link de reseña de Google' : 'URL' }}
        </label>
        <input
          :value="block.url ?? ''"
          placeholder="https://…"
          class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          @input="onType({ url: ($event.target as HTMLInputElement).value })"
        />
        <p v-if="block.block_type === 'google_review'" class="text-[11px] text-gray-400 mt-1">
          Búscalo en Google Maps → tu negocio → "Pedir reseñas" te da este link, o genera uno con la
          <a href="https://developers.google.com/maps/documentation/places/web-service/place-id" target="_blank" rel="noopener" class="text-indigo-600 hover:underline">herramienta de Place ID de Google</a>.
        </p>
      </div>
      <div v-if="block.block_type === 'text'">
        <label class="block text-xs font-medium text-gray-600 mb-1">Descripción</label>
        <textarea
          :value="block.description ?? ''"
          rows="2"
          class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          @input="onType({ description: ($event.target as HTMLTextAreaElement).value })"
        />
      </div>
      <div v-if="isBrandBlock">
        <label class="block text-xs font-medium text-gray-600 mb-1">Color del botón</label>
        <p class="text-[11px] text-gray-400 mb-1.5">
          Por defecto usa el mismo color que el resto de tus botones (defínelo en "Diseño").
        </p>
        <div class="flex gap-1.5 mb-2">
          <button
            type="button"
            class="flex-1 rounded-lg border-2 px-2 py-1.5 text-[11px] font-medium transition"
            :class="colorMode === 'theme' ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-500 hover:border-gray-300'"
            @click="setColorMode('theme')"
          >
            Color del tema
          </button>
          <button
            type="button"
            class="flex-1 rounded-lg border-2 px-2 py-1.5 text-[11px] font-medium transition"
            :class="colorMode === 'brand' ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-500 hover:border-gray-300'"
            @click="setColorMode('brand')"
          >
            Color de marca
          </button>
          <button
            type="button"
            class="flex-1 rounded-lg border-2 px-2 py-1.5 text-[11px] font-medium transition"
            :class="colorMode === 'custom' ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-500 hover:border-gray-300'"
            @click="setColorMode('custom')"
          >
            Personalizado
          </button>
        </div>
        <ColorInput v-if="colorMode === 'custom'" :model-value="customColor" @update:model-value="setCustomColor" />
      </div>
    </div>
  </div>
</template>
