<script setup lang="ts">
import { computed, watch } from 'vue'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'
import BlockRenderer from './BlockRenderer.vue'
import { ensureGoogleFontLoaded } from '@/composables/useGoogleFont'

const props = defineProps<{
  profile: (Pick<Profile, 'business_name' | 'description'> & { logo_url?: string | null }) | null
  theme: ProfileTheme | null
  blocks: ProfileBlock[]
}>()

const emit = defineEmits<{ blockClick: [id: string] }>()

watch(
  () => props.theme?.font_family,
  (family) => {
    if (family) ensureGoogleFontLoaded(family)
  },
  { immediate: true },
)

const backgroundStyle = computed(() => {
  if (!props.theme) return {}
  if (props.theme.background_type === 'image') {
    if (!props.theme.background_url) return { backgroundColor: '#f3f4f6' }
    return {
      backgroundImage: `url(${props.theme.background_url})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
    }
  }
  // 'gradient' and 'pattern' values are already full CSS `background`
  // shorthand (solid colors also work fine passed through `background`).
  return { background: props.theme.background_value }
})

const visibleBlocks = computed(() => props.blocks.filter((b) => b.is_visible))

const showLogoImage = computed(
  () => (props.theme?.logo_display_mode ?? 'initial') === 'image' && !!props.profile?.logo_url,
)

const logoShapeClass = computed(() => {
  switch (props.theme?.logo_shape) {
    case 'square':
      return 'rounded-none'
    case 'rounded':
      return 'rounded-xl'
    default:
      return 'rounded-full'
  }
})
</script>

<template>
  <div
    class="w-full h-full overflow-y-auto"
    :style="{ ...backgroundStyle, fontFamily: theme?.font_family || 'Inter', color: theme?.text_color }"
  >
    <div class="max-w-sm mx-auto px-4 py-8 flex flex-col items-center">
      <div class="animate-fade-in-up flex flex-col items-center">
        <img
          v-if="showLogoImage"
          :src="profile!.logo_url!"
          alt="Logo"
          class="w-20 h-20 object-cover shrink-0 shadow-sm"
          :class="logoShapeClass"
        />
        <div
          v-else
          class="w-20 h-20 flex items-center justify-center text-2xl font-semibold shrink-0"
          :class="logoShapeClass"
          :style="{ backgroundColor: theme?.logo_background_color, color: theme?.logo_text_color || '#ffffff' }"
        >
          {{ (profile?.business_name || '?').charAt(0).toUpperCase() }}
        </div>
        <h1 class="mt-3 text-lg font-semibold text-center">{{ profile?.business_name || 'Tu negocio' }}</h1>
        <p v-if="profile?.description" class="text-sm text-center opacity-80 mt-1">{{ profile.description }}</p>
      </div>

      <TransitionGroup
        tag="div"
        name="block-fade"
        class="w-full mt-6"
        :class="theme?.layout === 'grid' ? 'grid grid-cols-2 gap-3' : 'space-y-2.5'"
      >
        <BlockRenderer
          v-for="(block, i) in visibleBlocks"
          :key="block.id"
          :block="block"
          :theme="theme"
          :card="theme?.layout === 'grid'"
          :style="{ transitionDelay: `${Math.min(i, 8) * 40}ms` }"
          @click="emit('blockClick', block.id)"
        />
      </TransitionGroup>
      <p v-if="visibleBlocks.length === 0" class="text-center text-sm opacity-60 py-8">
        Todavía no hay bloques visibles.
      </p>
    </div>
  </div>
</template>
