<script setup lang="ts">
import { computed } from 'vue'
import { Star } from '@lucide/vue'
import type { ProfileTheme } from '@/types'
import { hexToRgba } from '@/composables/gradientUtils'

interface Testimonial {
  author: string
  quote: string
  rating?: number
}

const props = defineProps<{ content: Record<string, unknown> | null; theme: ProfileTheme | null }>()

const items = computed<Testimonial[]>(() => {
  const raw = props.content?.items
  return Array.isArray(raw) ? (raw as Testimonial[]) : []
})
</script>

<template>
  <div v-if="items.length" class="w-full space-y-2">
    <div
      v-for="(t, i) in items"
      :key="i"
      class="rounded-xl px-4 py-3 text-sm"
      :style="{ color: theme?.text_color, backgroundColor: hexToRgba(theme?.card_color ?? '#000000', theme?.card_opacity ?? 0.04) }"
    >
      <div v-if="t.rating" class="flex gap-0.5 mb-1">
        <Star
          v-for="n in 5"
          :key="n"
          :size="13"
          :fill="n <= (t.rating ?? 0) ? 'currentColor' : 'none'"
          :style="{ color: theme?.secondary_color ?? '#f59e0b' }"
        />
      </div>
      <p class="italic opacity-90">"{{ t.quote }}"</p>
      <p class="text-xs opacity-60 mt-1">— {{ t.author }}</p>
    </div>
  </div>
</template>
