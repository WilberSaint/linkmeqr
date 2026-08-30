<script setup lang="ts">
import { computed } from 'vue'
import {
  Phone, Mail, MapPin, Globe, Menu as MenuIcon, ShoppingBag,
  Image as ImageIcon, Video, Type, Link as LinkIcon, Clock, Quote, Map as MapIcon,
} from '@lucide/vue'
import type { ProfileBlock, ProfileTheme } from '@/types'
import { blockLabel } from '@/composables/blockLabels'
import { buttonShapeClass } from '@/composables/buttonStyle'
import InstagramIcon from '@/components/icons/InstagramIcon.vue'
import FacebookIcon from '@/components/icons/FacebookIcon.vue'
import WhatsappIcon from '@/components/icons/WhatsappIcon.vue'
import TiktokIcon from '@/components/icons/TiktokIcon.vue'
import YoutubeIcon from '@/components/icons/YoutubeIcon.vue'
import GalleryBlock from './blocks/GalleryBlock.vue'
import HoursBlock from './blocks/HoursBlock.vue'
import TestimonialsBlock from './blocks/TestimonialsBlock.vue'
import MapBlock from './blocks/MapBlock.vue'
import GoogleReviewBlock from './blocks/GoogleReviewBlock.vue'

const props = withDefaults(defineProps<{ block: ProfileBlock; theme: ProfileTheme | null; card?: boolean }>(), {
  card: false,
})
const emit = defineEmits<{ click: [] }>()

// Brand-accurate SVG icon + its official background color, used when the
// block doesn't override style (style_overrides.icon_color).
const brandBlocks: Record<string, { icon: unknown; brandColor: string }> = {
  instagram: { icon: InstagramIcon, brandColor: '#E1306C' },
  facebook: { icon: FacebookIcon, brandColor: '#1877F2' },
  whatsapp: { icon: WhatsappIcon, brandColor: '#25D366' },
  tiktok: { icon: TiktokIcon, brandColor: '#010101' },
  youtube: { icon: YoutubeIcon, brandColor: '#FF0000' },
}

const genericIconMap: Record<string, unknown> = {
  phone: Phone,
  email: Mail,
  location: MapPin,
  website: Globe,
  menu: MenuIcon,
  catalog: ShoppingBag,
  image: ImageIcon,
  video: Video,
  text: Type,
  link: LinkIcon,
  gallery: ImageIcon,
  hours: Clock,
  testimonials: Quote,
  map: MapIcon,
}

const isBrandBlock = computed(() => props.block.block_type in brandBlocks)
const icon = computed(() => brandBlocks[props.block.block_type]?.icon ?? genericIconMap[props.block.block_type] ?? LinkIcon)

function styleOverride(): { icon_color?: string; use_brand_color?: boolean } {
  return (props.block.style_overrides as { icon_color?: string; use_brand_color?: boolean } | null) ?? {}
}

// All buttons share the theme's button color by default, for a consistent
// look across the profile — brand colors (Instagram pink, WhatsApp green...)
// are opt-in per block via style_overrides.use_brand_color, not the default.
const buttonColor = computed(() => {
  const override = styleOverride()
  if (override.icon_color) return override.icon_color
  if (isBrandBlock.value && override.use_brand_color === true) {
    return brandBlocks[props.block.block_type].brandColor
  }
  return props.theme?.secondary_color ?? '#6366f1'
})

const buttonStyleClass = computed(() => buttonShapeClass(props.theme?.button_style))

/**
 * The block's destination, repaired if needed.
 *
 * The backend now normalizes URLs on save, but blocks stored before that —
 * a bare "www.cafemani.com" — would still resolve as a path relative to
 * /p/:slug and strand the visitor inside the site. Fixing it here too means
 * existing data behaves correctly without anyone having to re-save it.
 */
const resolvedHref = computed(() => {
  const raw = (props.block.media_url || props.block.url || '').trim()
  if (!raw) return ''
  if (/^[a-z][a-z0-9+.-]*:/i.test(raw) || raw.startsWith('//') || raw.startsWith('/')) return raw
  return raw.includes('.') ? `https://${raw}` : raw
})

/** tel:/mailto: must stay in place; only web destinations open a new tab. */
const isExternal = computed(() => /^(https?:)?\/\//i.test(resolvedHref.value))

function onClick() {
  // Navigation is the anchor's job now — this only records the click.
  emit('click')
}

const displayLabel = computed(() => props.block.title || blockLabel(props.block.block_type))
</script>

<template>
  <button
    v-if="block.block_type === 'text'"
    type="button"
    class="w-full text-left px-4 py-3 text-sm"
    :class="{ 'col-span-2': card }"
    :style="{ color: theme?.text_color }"
  >
    <p class="font-medium">{{ block.title }}</p>
    <p v-if="block.description" class="text-sm opacity-80 mt-0.5">{{ block.description }}</p>
  </button>

  <div v-else-if="block.block_type === 'google_review'" class="w-full" :class="{ 'col-span-2': card }">
    <GoogleReviewBlock :block="block" :theme="theme" @click="emit('click')" />
  </div>

  <div v-else-if="['gallery', 'hours', 'testimonials', 'map'].includes(block.block_type)" class="w-full" :class="{ 'col-span-2': card }">
    <p v-if="block.title" class="text-xs font-semibold uppercase tracking-wide opacity-60 mb-1.5" :style="{ color: theme?.text_color }">
      {{ block.title }}
    </p>
    <GalleryBlock v-if="block.block_type === 'gallery'" :content="block.content" :theme="theme" />
    <HoursBlock v-else-if="block.block_type === 'hours'" :content="block.content" :theme="theme" />
    <TestimonialsBlock v-else-if="block.block_type === 'testimonials'" :content="block.content" :theme="theme" />
    <MapBlock v-else-if="block.block_type === 'map'" :content="block.content" :theme="theme" />
  </div>

  <!-- A real <a> whenever there's somewhere to go: long-press to copy or
       open in a new tab, "link" announced to screen readers with its
       destination, no popup blocker in the way, and a crawlable href. Falls
       back to <button> only when the block genuinely has no target. -->
  <component
    :is="resolvedHref ? 'a' : 'button'"
    v-else
    :href="resolvedHref || undefined"
    :target="isExternal ? '_blank' : undefined"
    :rel="isExternal ? 'noopener noreferrer' : undefined"
    :type="resolvedHref ? undefined : 'button'"
    class="relative shadow-sm transition-all duration-150 ease-out hover:-translate-y-0.5 hover:shadow-md hover:brightness-105 active:translate-y-0 active:scale-[0.98] active:shadow-sm active:brightness-95"
    :class="[
      buttonStyleClass,
      card ? 'flex flex-col items-center justify-center gap-2 p-4 aspect-square text-center' : 'w-full flex items-center gap-3 px-4 py-3.5',
    ]"
    :style="{
      backgroundColor: theme?.button_style === 'outline' ? 'transparent' : buttonColor,
      borderColor: buttonColor,
      color: theme?.button_style === 'outline' ? buttonColor : theme?.button_text_color ?? '#ffffff',
    }"
    @click="onClick"
  >
    <span
      class="flex items-center justify-center rounded-full shrink-0"
      :class="card ? 'w-10 h-10' : 'w-9 h-9'"
      :style="{ backgroundColor: theme?.button_style === 'outline' ? 'transparent' : 'rgba(255,255,255,0.2)' }"
    >
      <component :is="icon" :size="card ? 20 : 22" color="currentColor" />
    </span>
    <span class="text-sm font-medium" :class="card ? 'line-clamp-2' : 'flex-1 text-center pr-9'">{{ displayLabel }}</span>
  </component>
</template>
