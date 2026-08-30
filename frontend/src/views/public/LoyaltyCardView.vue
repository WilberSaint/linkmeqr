<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Check, Gift, Stamp, Wallet } from '@lucide/vue'
import * as loyaltyApi from '@/api/loyalty'
import type { LoyaltyCardStatus, WalletSaveInfo } from '@/api/loyalty'

const route = useRoute()
const token = route.params.token as string

const status = ref<LoyaltyCardStatus | null>(null)
const loading = ref(true)
const notFound = ref(false)
const walletInfo = ref<WalletSaveInfo | null>(null)

const form = ref({ full_name: '', phone: '' })
const submitting = ref(false)
const formError = ref('')

// Fire-and-forget: the "Agregar a Google Wallet" button only appears once
// this resolves with enabled:true, so a failure here should just leave the
// button hidden rather than blocking the loyalty card itself.
async function loadWallet() {
  try {
    walletInfo.value = await loyaltyApi.getWalletSaveInfo(token)
  } catch {
    walletInfo.value = null
  }
}

async function load() {
  loading.value = true
  notFound.value = false
  try {
    status.value = await loyaltyApi.getCardStatus(token)
    if (status.value && !status.value.needs_registration) void loadWallet()
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

async function onRegister() {
  formError.value = ''
  submitting.value = true
  try {
    status.value = {
      ...(await loyaltyApi.registerCard(token, {
        full_name: form.value.full_name,
        phone: form.value.phone || undefined,
      })),
      needs_registration: false,
      just_stamped: true,
    }
    void loadWallet()
  } catch {
    formError.value = 'No se pudo registrar tu tarjeta. Intenta de nuevo.'
  } finally {
    submitting.value = false
  }
}

const stamps = computed(() => {
  if (!status.value) return []
  const count = status.value.stamps_count ?? 0
  return Array.from({ length: status.value.stamps_required }, (_, i) => i < count)
})

const isComplete = computed(() => (status.value?.stamps_count ?? 0) >= (status.value?.stamps_required ?? Infinity))

// The final reward always takes visual precedence over the mid one — once a
// card is complete, showing "you also got your mid reward" is just noise.
const midReached = computed(() => {
  if (!status.value || status.value.mid_reward_stamps == null || isComplete.value) return false
  return (status.value.stamps_count ?? 0) >= status.value.mid_reward_stamps
})

// The "N sellos más para..." progress line targets whichever tier is next:
// the mid reward if it exists and hasn't been reached yet, otherwise the
// final one (today's only behavior before mid rewards existed).
const nextTier = computed(() => {
  if (!status.value) return null
  const count = status.value.stamps_count ?? 0
  if (status.value.mid_reward_stamps != null && count < status.value.mid_reward_stamps) {
    return { stamps: status.value.mid_reward_stamps, description: status.value.mid_reward_description }
  }
  return { stamps: status.value.stamps_required, description: status.value.reward_description }
})

// Same theming convention as ProfilePreview.vue / BlockRenderer.vue: direct
// per-element :style bindings off the business's theme, not CSS variables.
const theme = computed(() => status.value?.theme ?? null)
const accentColor = computed(() => theme.value?.secondary_color || '#4f46e5')
const showLogoImage = computed(() => (theme.value?.logo_display_mode ?? 'initial') === 'image' && !!status.value?.logo_url)
const logoShapeClass = computed(() => {
  switch (theme.value?.logo_shape) {
    case 'square':
      return 'rounded-none'
    case 'rounded':
      return 'rounded-xl'
    default:
      return 'rounded-full'
  }
})
const buttonShapeClass = computed(() => {
  switch (theme.value?.button_style) {
    case 'square':
      return 'rounded-none'
    case 'pill':
      return 'rounded-full'
    case 'outline':
      return 'rounded-lg border-2 bg-transparent'
    default:
      return 'rounded-xl'
  }
})

onMounted(load)
</script>

<template>
  <div class="min-h-screen bg-gradient-to-b from-indigo-50 to-white flex items-center justify-center p-4">
    <div class="w-full max-w-sm">
      <div v-if="loading" class="text-center text-sm text-gray-400 py-16">Cargando…</div>

      <div v-else-if="notFound" class="bg-white rounded-2xl shadow-sm border border-gray-100 p-8 text-center">
        <p class="text-sm text-gray-500">Este programa de lealtad no existe o ya no está disponible.</p>
      </div>

      <div v-else-if="status && !status.is_active && status.needs_registration" class="bg-white rounded-2xl shadow-sm border border-gray-100 p-8 text-center">
        <p class="text-sm font-medium text-gray-900">{{ status.business_name }}</p>
        <p class="text-sm text-gray-500 mt-2">El programa de lealtad no está activo por el momento.</p>
      </div>

      <div v-else-if="status?.needs_registration" class="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
        <img
          v-if="showLogoImage"
          :src="status.logo_url!"
          alt="Logo"
          class="w-12 h-12 object-cover shrink-0 shadow-sm mx-auto mb-3"
          :class="logoShapeClass"
        />
        <div
          v-else
          class="w-12 h-12 flex items-center justify-center mx-auto mb-3"
          :class="logoShapeClass"
          :style="{ backgroundColor: `${accentColor}1a`, color: accentColor }"
        >
          <Stamp :size="22" />
        </div>
        <h1 class="text-base font-semibold text-gray-900 text-center">{{ status.business_name }}</h1>
        <p class="text-sm text-gray-500 text-center mt-1 mb-5">
          Regístrate para empezar a coleccionar sellos<template v-if="status.reward_description"> y ganar: <strong>{{ status.reward_description }}</strong></template>.
        </p>
        <form class="space-y-3" @submit.prevent="onRegister">
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Tu nombre</label>
            <input v-model="form.full_name" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Teléfono (opcional)</label>
            <input v-model="form.phone" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
          <button
            type="submit"
            class="w-full flex items-center justify-center px-3.5 py-2 text-sm font-medium transition disabled:opacity-50"
            :class="buttonShapeClass"
            :disabled="submitting"
            :style="{
              backgroundColor: theme?.button_style === 'outline' ? 'transparent' : accentColor,
              borderColor: accentColor,
              color: theme?.button_style === 'outline' ? accentColor : theme?.button_text_color || '#ffffff',
            }"
          >
            {{ submitting ? 'Registrando…' : 'Empezar a coleccionar sellos' }}
          </button>
        </form>
      </div>

      <div v-else-if="status" class="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 text-center">
        <p v-if="status.just_stamped" class="inline-flex items-center gap-1.5 text-xs font-medium text-green-700 bg-green-50 rounded-full px-3 py-1 mb-3">
          <Check :size="13" /> ¡Sello agregado!
        </p>
        <p v-else-if="!status.is_active" class="text-xs font-medium text-amber-700 bg-amber-50 rounded-full px-3 py-1 mb-3 inline-block">
          El programa está en pausa — por ahora no se agregan sellos nuevos.
        </p>
        <img
          v-if="showLogoImage"
          :src="status.logo_url!"
          alt="Logo"
          class="w-10 h-10 object-cover shrink-0 shadow-sm mx-auto mb-2"
          :class="logoShapeClass"
        />
        <h1 class="text-base font-semibold text-gray-900">{{ status.business_name }}</h1>
        <p class="text-sm text-gray-500 mt-1">{{ status.full_name }}</p>

        <div class="grid grid-cols-5 gap-2.5 my-6">
          <div
            v-for="(filled, i) in stamps"
            :key="i"
            class="aspect-square rounded-full flex items-center justify-center border-2"
            :class="status.mid_reward_stamps != null && i === status.mid_reward_stamps - 1 ? 'ring-2 ring-amber-400 ring-offset-1' : ''"
            :style="filled
              ? { backgroundColor: accentColor, borderColor: accentColor, color: theme?.button_text_color || '#ffffff' }
              : { borderStyle: 'dashed', borderColor: '#d1d5db', color: '#d1d5db' }"
          >
            <Stamp v-if="filled" :size="16" />
            <span v-else class="text-xs">{{ i + 1 }}</span>
          </div>
        </div>

        <div v-if="isComplete" class="rounded-xl bg-amber-50 border border-amber-200 px-4 py-3 flex items-center gap-2.5">
          <Gift :size="20" class="text-amber-600 shrink-0" />
          <p class="text-sm text-amber-800 text-left">
            ¡Completaste tu tarjeta! Muéstrale esta pantalla al negocio para canjear
            <strong v-if="status.reward_description">{{ status.reward_description }}</strong>
            <span v-else>tu premio</span>.
          </p>
        </div>
        <div v-else-if="midReached" class="rounded-xl bg-indigo-50 border border-indigo-200 px-4 py-3 flex items-center gap-2.5">
          <Gift :size="20" class="text-indigo-600 shrink-0" />
          <p class="text-sm text-indigo-800 text-left">
            ¡Alcanzaste tu premio intermedio! Muéstrale esta pantalla al negocio para canjear
            <strong v-if="status.mid_reward_description">{{ status.mid_reward_description }}</strong>
            <span v-else>tu premio</span>.
          </p>
        </div>
        <p v-else-if="nextTier" class="text-sm text-gray-500">
          {{ nextTier.stamps - (status.stamps_count ?? 0) }} sello(s) más para
          <strong v-if="nextTier.description">{{ nextTier.description }}</strong>
          <span v-else>tu premio</span>.
        </p>

        <a
          v-if="walletInfo?.enabled && walletInfo.save_url"
          :href="walletInfo.save_url"
          target="_blank"
          rel="noopener"
          class="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-gray-800"
        >
          <Wallet :size="16" /> Agregar a Google Wallet
        </a>
      </div>
    </div>
  </div>
</template>
