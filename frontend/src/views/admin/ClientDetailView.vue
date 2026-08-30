<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Printer, UserRound } from '@lucide/vue'
import type { DurationType, LicenseActivation, Profile, User } from '@/types'
import * as clientsApi from '@/api/clients'
import * as licensesApi from '@/api/licenses'
import { useAuthStore } from '@/stores/auth'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'

const durationOptions: { value: DurationType; label: string }[] = [
  { value: '1_MONTH', label: '1 mes' },
  { value: '3_MONTHS', label: '3 meses' },
  { value: '6_MONTHS', label: '6 meses' },
  { value: '1_YEAR', label: '1 año' },
  { value: 'CUSTOM', label: 'Personalizado' },
]

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const clientId = route.params.id as string

const client = ref<User | null>(null)
const profile = ref<Profile | null>(null)
const history = ref<LicenseActivation[]>([])
const loading = ref(true)
const impersonating = ref(false)

const profileForm = ref({ business_name: '', slug: '' })
const creatingProfile = ref(false)
const createError = ref('')

const licenseForm = ref<{ duration_type: DurationType; custom_days: number }>({
  duration_type: '1_MONTH',
  custom_days: 30,
})
const licenseSaving = ref(false)
const licenseError = ref('')
const licenseSuccess = ref('')

async function onActivateLicense() {
  licenseError.value = ''
  licenseSuccess.value = ''
  licenseSaving.value = true
  try {
    await clientsApi.activateLicenseForClient(clientId, licenseForm.value)
    licenseSuccess.value = 'Licencia activada correctamente.'
    history.value = await licensesApi.clientLicenseHistory(clientId)
  } catch {
    licenseError.value = 'No se pudo activar la licencia.'
  } finally {
    licenseSaving.value = false
  }
}

async function load() {
  loading.value = true
  try {
    client.value = await clientsApi.getClient(clientId)
    history.value = await licensesApi.clientLicenseHistory(clientId)
    try {
      profile.value = await clientsApi.getClientProfile(clientId)
    } catch {
      profile.value = null
    }
  } finally {
    loading.value = false
  }
}

async function onCreateProfile() {
  createError.value = ''
  creatingProfile.value = true
  try {
    profile.value = await clientsApi.createClientProfile(clientId, {
      business_name: profileForm.value.business_name,
      slug: profileForm.value.slug,
    })
  } catch {
    createError.value = 'No se pudo crear el perfil (¿slug ya en uso?).'
  } finally {
    creatingProfile.value = false
  }
}

async function onImpersonate() {
  impersonating.value = true
  try {
    const res = await clientsApi.impersonateClient(clientId)
    auth.startImpersonation(res.access_token, res.user)
    router.push({ name: 'client-dashboard' })
  } finally {
    impersonating.value = false
  }
}

function formatDate(d: string | null) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('es-MX', { year: 'numeric', month: 'short', day: 'numeric' })
}

onMounted(load)
</script>

<template>
  <div class="p-4 sm:p-6 max-w-3xl space-y-6">
    <div class="flex items-center justify-between">
      <RouterLink :to="{ name: 'admin-clients' }" class="inline-flex items-center gap-1 text-sm text-indigo-600 hover:underline">
        <ArrowLeft :size="14" /> Clientes
      </RouterLink>
      <div class="flex items-center gap-2">
        <RouterLink
          :to="{ name: 'admin-print-cards', params: { id: clientId } }"
          class="inline-flex items-center gap-1.5 rounded-lg px-3.5 py-2 text-sm font-medium bg-white text-gray-700 border border-gray-300 hover:bg-gray-50 transition"
        >
          <Printer :size="15" /> LinkMeQR Studio
        </RouterLink>
        <AppButton v-if="client?.is_active" variant="secondary" :disabled="impersonating" @click="onImpersonate">
          <UserRound :size="15" /> Ver como cliente
        </AppButton>
      </div>
    </div>

    <div v-if="client" class="flex items-center gap-3">
      <div class="w-11 h-11 rounded-full bg-indigo-100 text-indigo-700 font-semibold flex items-center justify-center shrink-0">
        {{ client.full_name.charAt(0).toUpperCase() }}
      </div>
      <div>
        <h1 class="text-lg font-semibold text-gray-900">{{ client.full_name }}</h1>
        <p class="text-sm text-gray-500">{{ client.email }}</p>
      </div>
      <AppBadge class="ml-auto" :tone="client.is_active ? 'green' : 'red'">{{ client.is_active ? 'Activa' : 'Inactiva' }}</AppBadge>
    </div>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Perfil digital</h2>

      <div v-if="profile" class="text-sm space-y-1">
        <p><span class="text-gray-500">Slug:</span> <code class="text-gray-900">/p/{{ profile.slug }}</code></p>
        <p><span class="text-gray-500">Nombre del negocio:</span> {{ profile.business_name }}</p>
        <p>
          <span class="text-gray-500">Publicado:</span>
          <AppBadge :tone="profile.is_published ? 'green' : 'gray'">{{ profile.is_published ? 'Sí' : 'No' }}</AppBadge>
        </p>
      </div>

      <form v-else class="space-y-3" @submit.prevent="onCreateProfile">
        <p class="text-sm text-gray-500 mb-2">Este cliente aún no tiene un perfil digital asignado.</p>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Nombre del negocio</label>
          <input v-model="profileForm.business_name" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Slug deseado (opcional)</label>
          <input v-model="profileForm.slug" placeholder="mi-negocio" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <p v-if="createError" class="text-sm text-red-600">{{ createError }}</p>
        <AppButton type="submit" :disabled="creatingProfile">Crear perfil</AppButton>
      </form>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Activar licencia</h2>
      <p class="text-xs text-gray-500 mb-3">
        Genera un código y lo activa de inmediato para este cliente. Si ya tiene una licencia
        vigente, la duración se sumará a su fecha de vencimiento actual.
      </p>
      <form class="flex flex-wrap items-end gap-3" @submit.prevent="onActivateLicense">
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Duración</label>
          <select v-model="licenseForm.duration_type" class="rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option v-for="d in durationOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
          </select>
        </div>
        <div v-if="licenseForm.duration_type === 'CUSTOM'">
          <label class="block text-xs font-medium text-gray-700 mb-1">Días</label>
          <input v-model.number="licenseForm.custom_days" type="number" min="1" class="w-24 rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <AppButton type="submit" :disabled="licenseSaving">Activar</AppButton>
      </form>
      <p v-if="licenseError" class="text-sm text-red-600 mt-2">{{ licenseError }}</p>
      <p v-if="licenseSuccess" class="text-sm text-green-600 mt-2">{{ licenseSuccess }}</p>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Historial de licencia</h2>
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
            <tr v-if="!loading && history.length === 0">
              <td colspan="4" class="py-6 text-center text-gray-400">Sin activaciones todavía.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
