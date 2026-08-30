<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { LicenseActivation } from '@/types'
import { useLicenseStore } from '@/stores/license'
import * as licensesApi from '@/api/licenses'
import { licenseStatusLabel } from '@/composables/licenseLabels'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'

const licenseStore = useLicenseStore()
const history = ref<LicenseActivation[]>([])
const code = ref('')
const activating = ref(false)
const error = ref('')
const success = ref('')

async function load() {
  await licenseStore.refresh()
  history.value = await licensesApi.myLicenseHistory()
}

async function onActivate() {
  error.value = ''
  success.value = ''
  activating.value = true
  try {
    await licensesApi.activateCode(code.value.trim())
    success.value = '¡Listo! Tu licencia se extendió.'
    code.value = ''
    await load()
  } catch (e) {
    // The API already distinguishes "no existe" from "ya se usó" from "es de
    // otra cuenta"; collapsing all three into one message left people retyping
    // a code that was never going to work for them.
    const body = (e as { response?: { data?: { message?: string } } })?.response?.data
    error.value = body?.message || 'No pudimos activar ese código. Revísalo e inténtalo de nuevo.'
  } finally {
    activating.value = false
  }
}

/** Below this, the expiry stops being a date and starts being a problem. */
const EXPIRY_WARNING_DAYS = 15

const daysLeft = computed(() => licenseStore.license?.days_remaining ?? null)
const isExpiring = computed(
  () => licenseStore.license?.status === 'ACTIVE' && daysLeft.value !== null && daysLeft.value <= EXPIRY_WARNING_DAYS,
)

function formatDate(d: string | null) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('es-MX', { year: 'numeric', month: 'short', day: 'numeric' })
}

function statusTone(status?: string) {
  if (status === 'ACTIVE') return 'green'
  if (status === 'EXPIRED') return 'red'
  return 'gray'
}

onMounted(load)
</script>

<template>
  <div class="p-4 sm:p-6 max-w-2xl space-y-6">
    <AppPageHeader
      title="Mi licencia"
      description="Mientras tu licencia esté activa, tu perfil público y tus códigos QR funcionan con normalidad."
    />

    <section
      class="rounded-xl border p-5"
      :class="
        licenseStore.license?.status === 'ACTIVE'
          ? isExpiring
            ? 'border-amber-200 bg-amber-50'
            : 'border-green-200 bg-green-50'
          : 'border-red-200 bg-red-50'
      "
    >
      <div class="flex items-start justify-between gap-4">
        <div>
          <AppBadge :tone="statusTone(licenseStore.license?.status)">
            {{ licenseStatusLabel(licenseStore.license?.status) }}
          </AppBadge>
          <p v-if="daysLeft !== null" class="mt-2 text-2xl font-semibold tracking-tight text-gray-900">
            {{ daysLeft }} {{ daysLeft === 1 ? 'día' : 'días' }}
          </p>
          <p class="mt-0.5 text-sm text-gray-600">
            Vence el {{ formatDate(licenseStore.license?.expires_at ?? null) }}
          </p>
        </div>
      </div>
      <p v-if="isExpiring" class="mt-3 border-t border-amber-200 pt-3 text-sm text-amber-800">
        Tu licencia está por vencer. Pídele un código de renovación a quien te dio el servicio y actívalo aquí abajo.
      </p>
      <p v-else-if="licenseStore.license?.status !== 'ACTIVE'" class="mt-3 border-t border-red-200 pt-3 text-sm text-red-800">
        Mientras esté vencida, tu perfil público no se muestra a quien escanee tu QR.
      </p>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="mb-1 text-sm font-semibold text-gray-900">Activar un código</h2>
      <p class="mb-3 text-sm text-gray-500">
        Es el código que te entrega quien te vendió el servicio. Escríbelo tal cual y tu licencia se extiende al
        instante.
      </p>
      <form class="flex gap-2" @submit.prevent="onActivate">
        <input
          v-model="code"
          placeholder="XXXX-XXXX-XXXX-XXXX"
          required
          autocapitalize="characters"
          autocomplete="off"
          spellcheck="false"
          class="flex-1 rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm uppercase"
        />
        <AppButton type="submit" :disabled="activating || !code.trim()">
          {{ activating ? 'Activando…' : 'Activar' }}
        </AppButton>
      </form>
      <p v-if="error" class="text-sm text-red-600 mt-2">{{ error }}</p>
      <p v-if="success" class="text-sm text-green-600 mt-2">{{ success }}</p>
      <p class="mt-2 text-xs text-gray-400">
        Si aún te quedan días, los del código nuevo se suman: no pierdes nada por activarlo antes de tiempo.
      </p>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Historial de activaciones</h2>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="text-gray-500 text-xs uppercase">
            <tr>
              <th class="text-left py-1.5 pr-4 font-medium whitespace-nowrap">Activado</th>
              <th class="text-left py-1.5 pr-4 font-medium whitespace-nowrap">Días agregados</th>
              <th class="text-left py-1.5 pr-4 font-medium whitespace-nowrap">Vencimiento anterior</th>
              <th class="text-left py-1.5 font-medium whitespace-nowrap">Nuevo vencimiento</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-for="h in history" :key="h.id">
              <td class="py-1.5 pr-4 whitespace-nowrap">{{ formatDate(h.activated_at) }}</td>
              <td class="py-1.5 pr-4 whitespace-nowrap">{{ h.duration_days_added }}</td>
              <td class="py-1.5 pr-4 whitespace-nowrap">{{ formatDate(h.previous_expires_at) }}</td>
              <td class="py-1.5 font-medium text-gray-900 whitespace-nowrap">{{ formatDate(h.new_expires_at) }}</td>
            </tr>
            <tr v-if="history.length === 0">
              <td colspan="4" class="py-8 text-center text-sm text-gray-400">
                Aquí quedará registrada cada renovación que actives.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
