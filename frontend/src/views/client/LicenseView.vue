<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { LicenseActivation } from '@/types'
import { useLicenseStore } from '@/stores/license'
import * as licensesApi from '@/api/licenses'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'

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
    success.value = 'Código activado correctamente.'
    code.value = ''
    await load()
  } catch {
    error.value = 'Código inválido, ya usado o revocado.'
  } finally {
    activating.value = false
  }
}

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
  <div class="p-6 max-w-2xl space-y-6">
    <h1 class="text-lg font-semibold text-gray-900">Mi licencia</h1>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-semibold text-gray-900">Estado actual</h2>
        <AppBadge :tone="statusTone(licenseStore.license?.status)">
          {{ licenseStore.license?.status ?? 'INACTIVE' }}
        </AppBadge>
      </div>
      <div class="text-sm space-y-1 text-gray-600">
        <p>Vencimiento: <span class="text-gray-900">{{ formatDate(licenseStore.license?.expires_at ?? null) }}</span></p>
        <p v-if="licenseStore.license?.days_remaining !== null && licenseStore.license?.days_remaining !== undefined">
          Días restantes: <span class="text-gray-900 font-medium">{{ licenseStore.license?.days_remaining }}</span>
        </p>
      </div>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Activar código</h2>
      <form class="flex gap-2" @submit.prevent="onActivate">
        <input
          v-model="code"
          placeholder="XXXX-XXXX-XXXX-XXXX"
          required
          class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono uppercase"
        />
        <AppButton type="submit" :disabled="activating">Activar</AppButton>
      </form>
      <p v-if="error" class="text-sm text-red-600 mt-2">{{ error }}</p>
      <p v-if="success" class="text-sm text-green-600 mt-2">{{ success }}</p>
      <p class="text-xs text-gray-400 mt-2">
        Si ya tienes una licencia activa, la duración del nuevo código se sumará a tu fecha de vencimiento actual.
      </p>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Historial de activaciones</h2>
      <table class="w-full text-sm">
        <thead class="text-gray-500 text-xs uppercase">
          <tr>
            <th class="text-left py-1.5 font-medium">Activado</th>
            <th class="text-left py-1.5 font-medium">Días agregados</th>
            <th class="text-left py-1.5 font-medium">Vencimiento anterior</th>
            <th class="text-left py-1.5 font-medium">Nuevo vencimiento</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="h in history" :key="h.id">
            <td class="py-1.5">{{ formatDate(h.activated_at) }}</td>
            <td class="py-1.5">{{ h.duration_days_added }}</td>
            <td class="py-1.5">{{ formatDate(h.previous_expires_at) }}</td>
            <td class="py-1.5 font-medium text-gray-900">{{ formatDate(h.new_expires_at) }}</td>
          </tr>
          <tr v-if="history.length === 0">
            <td colspan="4" class="py-6 text-center text-gray-400">Sin activaciones todavía.</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
