<script setup lang="ts">
import { computed, ref } from 'vue'
import { X } from '@lucide/vue'
import * as profileApi from '@/api/profile'

interface GalleryImage {
  media_id: string
  file_path: string
  caption?: string
}

const props = defineProps<{ content: Record<string, unknown> | null }>()
const emit = defineEmits<{ update: [content: Record<string, unknown>] }>()

const uploading = ref(false)

const images = computed<GalleryImage[]>(() => {
  const raw = props.content?.images
  return Array.isArray(raw) ? (raw as GalleryImage[]) : []
})

async function onFilesSelected(e: Event) {
  const files = Array.from((e.target as HTMLInputElement).files ?? [])
  ;(e.target as HTMLInputElement).value = ''
  if (!files.length) return
  uploading.value = true
  try {
    const uploaded: GalleryImage[] = []
    for (const file of files) {
      const media = await profileApi.uploadMedia(file)
      uploaded.push({ media_id: media.id, file_path: media.file_path, caption: '' })
    }
    emit('update', { images: [...images.value, ...uploaded] })
  } finally {
    uploading.value = false
  }
}

function removeImage(i: number) {
  emit('update', { images: images.value.filter((_, idx) => idx !== i) })
}

function setCaption(i: number, caption: string) {
  const next = images.value.map((img, idx) => (idx === i ? { ...img, caption } : img))
  emit('update', { images: next })
}
</script>

<template>
  <div class="space-y-2">
    <label class="block text-xs font-medium text-gray-600 mb-1">Fotos</label>
    <div class="grid grid-cols-3 gap-2">
      <div v-for="(img, i) in images" :key="img.media_id" class="relative">
        <img :src="img.file_path" class="w-full aspect-square object-cover rounded-lg border border-gray-200" />
        <button
          type="button"
          class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-white border border-gray-300 flex items-center justify-center text-gray-500 hover:text-red-600"
          @click="removeImage(i)"
        >
          <X :size="12" />
        </button>
        <input
          :value="img.caption ?? ''"
          placeholder="Pie de foto"
          class="mt-1 w-full rounded border border-gray-200 px-1.5 py-1 text-[11px]"
          @change="setCaption(i, ($event.target as HTMLInputElement).value)"
        />
      </div>
      <label class="aspect-square rounded-lg border border-dashed border-gray-300 flex items-center justify-center text-[11px] text-gray-400 cursor-pointer hover:border-gray-400">
        {{ uploading ? '…' : '+ Agregar' }}
        <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple class="hidden" :disabled="uploading" @change="onFilesSelected" />
      </label>
    </div>
  </div>
</template>
