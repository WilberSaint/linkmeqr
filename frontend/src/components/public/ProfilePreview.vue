<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Share2, UserPlus } from '@lucide/vue'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'
import BlockRenderer from './BlockRenderer.vue'
import { ensureGoogleFontLoaded } from '@/composables/useGoogleFont'
import { buildVCard, downloadVCard, hasContactInfo } from '@/composables/vcard'

const props = defineProps<{
  profile:
    | (Pick<Profile, 'business_name' | 'description'> & { logo_url?: string | null; cover_url?: string | null })
    | null
  theme: ProfileTheme | null
  blocks: ProfileBlock[]
  /**
   * The page's own public address. Its presence is also what turns the
   * save-contact / share row on: the editor's live preview passes it so the
   * business sees exactly what a visitor gets, and both actions need a real
   * URL to be worth anything.
   */
  publicUrl?: string
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

    if (props.theme.background_fit === 'repeat') {
      // The natural-size tile a genuinely seamless pattern image is meant
      // to be viewed at — no crop, no letterbox, just repeats in both axes.
      return {
        backgroundImage: `url(${props.theme.background_url})`,
        backgroundRepeat: 'repeat',
        backgroundSize: 'auto',
      }
    }

    if (props.theme.background_fit === 'contain') {
      // Shows the whole image, never cropped — for a framed/poster-style
      // illustration whose own border would otherwise get cut off by
      // `cover` on a screen shaped differently than the source image.
      // Letterboxing uses the theme's own background_value (whatever
      // color/gradient was set before switching to 'image') as the fill
      // behind it, layered as a second background-image, so a design whose
      // own edges already match that color blends in with no visible seam.
      const fallback = props.theme.background_value.startsWith('#')
        ? `linear-gradient(${props.theme.background_value}, ${props.theme.background_value})`
        : props.theme.background_value
      return {
        backgroundImage: `url(${props.theme.background_url}), ${fallback}`,
        backgroundSize: 'contain, cover',
        backgroundRepeat: 'no-repeat, no-repeat',
        backgroundPosition: 'center, center',
      }
    }

    // 'cover': fills edge to edge, cropping whatever doesn't fit — right
    // for a full-bleed photo, wrong for a bordered illustration (see
    // 'contain' above).
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

// --- guardar contacto / compartir -------------------------------------

const showSaveContact = computed(() => Boolean(props.publicUrl) && hasContactInfo(visibleBlocks.value))
const showShare = computed(() => Boolean(props.publicUrl))
const shareFeedback = ref('')

function onSaveContact() {
  if (!props.profile || !props.publicUrl) return
  const vcard = buildVCard(props.profile, visibleBlocks.value, props.publicUrl)
  // Strip only what a filesystem actually rejects — an allow-list would
  // eat the accents and ñ that most business names here contain.
  const safeName =
    (props.profile.business_name || 'contacto').replace(/[/\\:*?"<>|]/g, '').trim() || 'contacto'
  downloadVCard(`${safeName}.vcf`, vcard)
}

async function onShare() {
  if (!props.publicUrl) return
  const title = props.profile?.business_name || 'LinkMeQR'
  // The native sheet is the whole point on a phone (WhatsApp is one tap in);
  // desktop browsers mostly lack it, so those fall back to the clipboard.
  if (navigator.share) {
    try {
      await navigator.share({ title, url: props.publicUrl })
      return
    } catch {
      // Dismissing the share sheet rejects — not an error worth surfacing.
      return
    }
  }
  try {
    await navigator.clipboard.writeText(props.publicUrl)
    shareFeedback.value = 'Enlace copiado'
    setTimeout(() => (shareFeedback.value = ''), 2000)
  } catch {
    shareFeedback.value = 'No se pudo copiar'
    setTimeout(() => (shareFeedback.value = ''), 2000)
  }
}

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
    <!-- Cover banner: full-bleed, outside the centered column, so it reads as
         part of the page rather than as one more card in the stack. -->
    <div v-if="profile?.cover_url" class="w-full h-36 sm:h-44 overflow-hidden">
      <img :src="profile.cover_url" alt="" class="w-full h-full object-cover" />
    </div>

    <div
      class="max-w-sm mx-auto px-4 pb-8 flex flex-col items-center"
      :class="profile?.cover_url ? 'pt-0' : 'pt-8'"
    >
      <div
        class="animate-fade-in-up flex flex-col items-center"
        :class="profile?.cover_url ? '-mt-10' : ''"
      >
        <img
          v-if="showLogoImage"
          :src="profile!.logo_url!"
          alt="Logo"
          class="w-20 h-20 object-cover shrink-0 shadow-sm"
          :class="[logoShapeClass, profile?.cover_url ? 'ring-4 ring-white' : '']"
        />
        <div
          v-else
          class="w-20 h-20 flex items-center justify-center text-2xl font-semibold shrink-0"
          :class="[logoShapeClass, profile?.cover_url ? 'ring-4 ring-white' : '']"
          :style="{ backgroundColor: theme?.logo_background_color, color: theme?.logo_text_color || '#ffffff' }"
        >
          {{ (profile?.business_name || '?').charAt(0).toUpperCase() }}
        </div>
        <h1 class="mt-3 text-lg font-semibold text-center">{{ profile?.business_name || 'Tu negocio' }}</h1>
        <p v-if="profile?.description" class="text-sm text-center opacity-80 mt-1">{{ profile.description }}</p>

        <div v-if="showSaveContact || showShare" class="flex items-center gap-2 mt-3">
          <button
            v-if="showSaveContact"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition hover:opacity-80"
            :style="{ borderColor: 'currentColor', color: theme?.text_color }"
            @click="onSaveContact"
          >
            <UserPlus class="w-3.5 h-3.5" />
            Guardar contacto
          </button>
          <button
            v-if="showShare"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition hover:opacity-80"
            :style="{ borderColor: 'currentColor', color: theme?.text_color }"
            @click="onShare"
          >
            <Share2 class="w-3.5 h-3.5" />
            {{ shareFeedback || 'Compartir' }}
          </button>
        </div>
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
