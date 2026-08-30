<script setup lang="ts">
import draggable from 'vuedraggable'
import type { BlockType, ProfileBlock } from '@/types'
import BlockListItem from './BlockListItem.vue'

const props = defineProps<{
  blocks: ProfileBlock[]
  uploadingMenuFileFor?: string | null
  /** Clicks per block id, so the list can show what each one is actually doing. */
  clicksByBlock?: Record<string, number>
}>()
const emit = defineEmits<{
  reorder: [blocks: ProfileBlock[]]
  update: [id: string, payload: Partial<ProfileBlock>, debounced?: boolean]
  remove: [id: string]
  duplicate: [id: string]
  add: [type: BlockType]
  uploadMenuFile: [id: string, file: File]
}>()

// Twenty block types in one flat list gave no clue what any of them were for,
// or that "Galería" and "Testimonios" even existed down at the bottom.
// Grouping them by what the visitor would DO turns scanning the list into a
// decision instead of a memory test.
const blockGroups: { label: string; types: { value: BlockType; label: string }[] }[] = [
  {
    label: 'Redes sociales',
    types: [
      { value: 'instagram', label: 'Instagram' },
      { value: 'facebook', label: 'Facebook' },
      { value: 'tiktok', label: 'TikTok' },
      { value: 'youtube', label: 'YouTube' },
    ],
  },
  {
    label: 'Contacto',
    types: [
      { value: 'whatsapp', label: 'WhatsApp' },
      { value: 'phone', label: 'Teléfono' },
      { value: 'email', label: 'Correo' },
      { value: 'location', label: 'Ubicación' },
      { value: 'map', label: 'Mapa' },
      { value: 'hours', label: 'Horario de atención' },
    ],
  },
  {
    label: 'Tu negocio',
    types: [
      { value: 'menu', label: 'Menú' },
      { value: 'catalog', label: 'Catálogo' },
      { value: 'website', label: 'Sitio web' },
      { value: 'google_review', label: 'Reseña en Google' },
      { value: 'testimonials', label: 'Testimonios' },
    ],
  },
  {
    label: 'Contenido',
    types: [
      { value: 'gallery', label: 'Galería de fotos' },
      { value: 'image', label: 'Imagen' },
      { value: 'video', label: 'Video' },
      { value: 'text', label: 'Texto' },
      { value: 'link', label: 'Enlace personalizado' },
    ],
  },
]

function onDragEnd(newList: ProfileBlock[]) {
  emit('reorder', newList)
}

/**
 * Reordering by drag is fiddly on a touch screen — the surface most business
 * owners actually edit their page on — so arrows on each row do the same
 * swap without a drag gesture. Both go through the same 'reorder' event, so
 * dragging and clicking the same page never disagree about how it works.
 */
function moveBlock(id: string, direction: -1 | 1) {
  const index = props.blocks.findIndex((b) => b.id === id)
  const target = index + direction
  if (index === -1 || target < 0 || target >= props.blocks.length) return
  const next = [...props.blocks]
  ;[next[index], next[target]] = [next[target], next[index]]
  emit('reorder', next)
}
</script>

<template>
  <div class="space-y-3">
    <select
      class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
      @change="
        (e) => {
          const v = (e.target as HTMLSelectElement).value as BlockType
          if (v) emit('add', v)
          ;(e.target as HTMLSelectElement).value = ''
        }
      "
    >
      <option value="">+ Agregar bloque…</option>
      <optgroup v-for="g in blockGroups" :key="g.label" :label="g.label">
        <option v-for="t in g.types" :key="t.value" :value="t.value">{{ t.label }}</option>
      </optgroup>
    </select>

    <draggable
      :model-value="blocks"
      item-key="id"
      handle=".drag-handle"
      class="space-y-2"
      @update:model-value="onDragEnd"
    >
      <template #item="{ element }">
        <BlockListItem
          :block="element"
          :clicks="props.clicksByBlock?.[element.id]"
          :menu-file-uploading="uploadingMenuFileFor === element.id"
          :can-move-up="blocks.findIndex((b) => b.id === element.id) > 0"
          :can-move-down="blocks.findIndex((b) => b.id === element.id) < blocks.length - 1"
          @update="(payload, debounced) => emit('update', element.id, payload, debounced)"
          @remove="emit('remove', element.id)"
          @duplicate="emit('duplicate', element.id)"
          @upload-menu-file="(file) => emit('uploadMenuFile', element.id, file)"
          @move="(direction) => moveBlock(element.id, direction)"
        />
      </template>
    </draggable>

    <div
      v-if="blocks.length === 0"
      class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center"
    >
      <p class="text-sm font-medium text-gray-700">Tu perfil aún no tiene bloques</p>
      <p class="mx-auto mt-1.5 max-w-xs text-xs text-gray-500">
        Cada bloque es un botón o sección de tu página: tu WhatsApp, tu menú, tus redes. Elige el primero en la lista
        de arriba.
      </p>
    </div>
  </div>
</template>
