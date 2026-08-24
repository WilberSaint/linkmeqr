<script setup lang="ts">
import draggable from 'vuedraggable'
import type { BlockType, ProfileBlock } from '@/types'
import BlockListItem from './BlockListItem.vue'

const props = defineProps<{ blocks: ProfileBlock[]; uploadingMenuFileFor?: string | null }>()
const emit = defineEmits<{
  reorder: [blocks: ProfileBlock[]]
  update: [id: string, payload: Partial<ProfileBlock>]
  remove: [id: string]
  duplicate: [id: string]
  add: [type: BlockType]
  uploadMenuFile: [id: string, file: File]
}>()

const blockTypes: { value: BlockType; label: string }[] = [
  { value: 'instagram', label: 'Instagram' },
  { value: 'facebook', label: 'Facebook' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'youtube', label: 'YouTube' },
  { value: 'whatsapp', label: 'WhatsApp' },
  { value: 'phone', label: 'Teléfono' },
  { value: 'email', label: 'Email' },
  { value: 'location', label: 'Ubicación' },
  { value: 'website', label: 'Sitio web' },
  { value: 'menu', label: 'Menú' },
  { value: 'catalog', label: 'Catálogo' },
  { value: 'image', label: 'Imagen' },
  { value: 'video', label: 'Video' },
  { value: 'text', label: 'Texto' },
  { value: 'link', label: 'Enlace personalizado' },
]

function onDragEnd(newList: ProfileBlock[]) {
  emit('reorder', newList)
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
      <option v-for="t in blockTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
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
          :menu-file-uploading="uploadingMenuFileFor === element.id"
          @update="(payload) => emit('update', element.id, payload)"
          @remove="emit('remove', element.id)"
          @duplicate="emit('duplicate', element.id)"
          @upload-menu-file="(file) => emit('uploadMenuFile', element.id, file)"
        />
      </template>
    </draggable>

    <p v-if="blocks.length === 0" class="text-center text-sm text-gray-400 py-6">
      Agrega tu primer bloque arriba.
    </p>
  </div>
</template>
