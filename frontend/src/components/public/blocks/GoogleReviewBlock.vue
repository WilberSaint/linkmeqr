<script setup lang="ts">
import { computed } from 'vue'
import { Star } from '@lucide/vue'
import type { ProfileBlock, ProfileTheme } from '@/types'
import GoogleIcon from '@/components/icons/GoogleIcon.vue'
import { hexToRgba } from '@/composables/gradientUtils'
import { buttonShapeClass } from '@/composables/buttonStyle'

const props = defineProps<{ block: ProfileBlock; theme: ProfileTheme | null }>()
const emit = defineEmits<{ click: [] }>()

// Same reasoning as BlockRenderer: a real link, repaired if it was stored
// without a scheme, so it can be long-pressed, opened in a tab and read
// correctly by a screen reader.
const href = computed(() => {
  const raw = (props.block.url ?? '').trim()
  if (!raw) return ''
  if (/^[a-z][a-z0-9+.-]*:/i.test(raw) || raw.startsWith('//') || raw.startsWith('/')) return raw
  return raw.includes('.') ? `https://${raw}` : raw
})

function onClick() {
  emit('click')
}

const shapeClass = computed(() => buttonShapeClass(props.theme?.button_style))
</script>

<template>
  <div
    class="w-full rounded-xl p-4 text-center flex flex-col items-center gap-2"
    :style="{ backgroundColor: hexToRgba(theme?.card_color ?? '#000000', theme?.card_opacity ?? 0.04), color: theme?.text_color }"
  >
    <GoogleIcon :size="22" />
    <p class="text-sm font-medium">{{ block.title || '¿Nos regalas una reseña?' }}</p>
    <div class="flex gap-0.5">
      <Star v-for="n in 5" :key="n" :size="16" fill="#FBBC05" color="#FBBC05" />
    </div>
    <component
      :is="href ? 'a' : 'button'"
      :href="href || undefined"
      :target="href ? '_blank' : undefined"
      :rel="href ? 'noopener noreferrer' : undefined"
      :type="href ? undefined : 'button'"
      class="mt-1 inline-block px-4 py-2 text-sm font-medium shadow-sm transition-all duration-150 ease-out hover:-translate-y-0.5 hover:shadow-md active:translate-y-0 active:scale-[0.98]"
      :class="shapeClass"
      :style="{
        backgroundColor: theme?.button_style === 'outline' ? 'transparent' : theme?.secondary_color ?? '#6366f1',
        borderColor: theme?.secondary_color ?? '#6366f1',
        color: theme?.button_style === 'outline' ? theme?.secondary_color : theme?.button_text_color ?? '#ffffff',
      }"
      @click="onClick"
    >
      Escribir reseña
    </component>
  </div>
</template>
