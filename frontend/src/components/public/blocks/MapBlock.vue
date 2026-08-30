<script setup lang="ts">
import { computed } from 'vue'
import type { ProfileTheme } from '@/types'

const props = defineProps<{ content: Record<string, unknown> | null; theme: ProfileTheme | null }>()

const embedUrl = computed(() => {
  const url = props.content?.embed_url
  if (typeof url !== 'string') return ''
  // Only ever iframe well-known map-embed origins — this URL is set by the
  // business owner themselves (same trust level as their background image
  // URL), but there's no reason to allow embedding arbitrary third-party pages.
  const allowed = ['https://www.google.com/maps', 'https://maps.google.com', 'https://www.openstreetmap.org']
  return allowed.some((p) => url.startsWith(p)) ? url : ''
})
</script>

<template>
  <div v-if="embedUrl" class="w-full rounded-xl overflow-hidden" style="aspect-ratio: 16 / 10">
    <iframe :src="embedUrl" class="w-full h-full border-0" loading="lazy" referrerpolicy="no-referrer-when-downgrade" />
  </div>
  <p v-else class="text-xs text-center opacity-60" :style="{ color: theme?.text_color }">
    Mapa no configurado.
  </p>
</template>
