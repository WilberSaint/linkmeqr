<script setup lang="ts">
import { computed } from 'vue'
import type { ProfileTheme } from '@/types'

interface GalleryImage {
  media_id: string
  file_path: string
  caption?: string
}

const props = defineProps<{ content: Record<string, unknown> | null; theme: ProfileTheme | null }>()

const images = computed<GalleryImage[]>(() => {
  const raw = props.content?.images
  return Array.isArray(raw) ? (raw as GalleryImage[]) : []
})
</script>

<template>
  <div v-if="images.length" class="grid grid-cols-3 gap-1.5">
    <a
      v-for="(img, i) in images"
      :key="i"
      :href="img.file_path"
      target="_blank"
      rel="noopener"
      class="aspect-square rounded-lg overflow-hidden bg-black/5"
      :title="img.caption"
    >
      <img :src="img.file_path" :alt="img.caption || ''" class="w-full h-full object-cover" />
    </a>
  </div>
  <p v-else class="text-xs text-center opacity-60" :style="{ color: theme?.text_color }">
    Galería sin fotos todavía.
  </p>
</template>
