<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { ClientWithLicense, DurationType } from '@/types'
import * as clientsApi from '@/api/clients'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'
import AppModal from '@/components/common/AppModal.vue'

const clients = ref<ClientWithLicense[]>([])
const loading = ref(false)
const showCreate = ref(false)

const form = ref({ email: '', password: '', full_name: '', phone: '' })
const formError = ref('')

const durationOptions: { value: DurationType; label: string }[] = [
  { value: '1_MONTH', label: '1 mes' },
  { value: '3_MONTHS', label: '3 meses' },
  { value: '6_MONTHS', label: '6 meses' },
  { value: '1_YEAR', label: '1 año' },
  { value: 'CUSTOM', label: 'Personalizado' },
]

const licenseTarget = ref<ClientWithLicense | null>(null)
const licenseForm = ref<{ duration_type: DurationType; custom_days: number }>({
  duration_type: '1_MONTH',
  custom_days: 30,
})
const licenseSaving = ref(false)
const licenseError = ref('')
const licenseSuccess = ref('')

async function load() {
  loading.value = true
  try {
    clients.value = await clientsApi.listClients()
  } finally {
    loading.value = false
  }
}

async function onCreate() {
  formError.value = ''
  try {
    await clientsApi.createClient({
      email: form.value.email,
      password: form.value.password,
      full_name: form.value.full_name,
      phone: form.value.phone || null,
    })
    showCreate.value = false
    form.value = { email: '', password: '', full_name: '', phone: '' }
    await load()
  } catch {
    formError.value = 'No se pudo crear el cliente (¿email duplicado?).'
  }
}

async function onToggleActive(client: ClientWithLicense) {
  if (client.is_active) {
    await clientsApi.deactivateClient(client.id)
  } else {
    await clientsApi.activateClient(client.id)
  }
  await load()
}

function openLicenseModal(client: ClientWithLicense) {
  licenseTarget.value = client
  licenseError.value = ''
  licenseSuccess.value = ''
  licenseForm.value = { duration_type: '1_MONTH', custom_days: 30 }
}

async function onActivateLicense() {
  if (!licenseTarget.value) return
  licenseError.value = ''
  licenseSuccess.value = ''
  licenseSaving.value = true
  try {
    await clientsApi.activateLicenseForClient(licenseTarget.value.id, licenseForm.value)
    licenseSuccess.value = 'Licencia activada correctamente.'
    await load()
  } catch {
    licenseError.value = 'No se pudo activar la licencia.'
  } finally {
    licenseSaving.value = false
  }
}

function licenseTone(status: string) {
  if (status === 'ACTIVE') return 'green'
  if (status === 'EXPIRED') return 'red'
  return 'gray'
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-5xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-lg font-semibold text-gray-900">Clientes</h1>
      <AppButton @click="showCreate = true">+ Nuevo cliente</AppButton>
    </div>

    <div class="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium">Nombre</th>
            <th class="text-left px-4 py-2.5 font-medium">Email</th>
            <th class="text-left px-4 py-2.5 font-medium">Cuenta</th>
            <th class="text-left px-4 py-2.5 font-medium">Licencia</th>
            <th class="text-right px-4 py-2.5 font-medium">Acciones</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="c in clients" :key="c.id">
            <td class="px-4 py-2.5 text-gray-900">{{ c.full_name }}</td>
            <td class="px-4 py-2.5 text-gray-600">{{ c.email }}</td>
            <td class="px-4 py-2.5">
              <AppBadge :tone="c.is_active ? 'green' : 'red'">
                {{ c.is_active ? 'Activa' : 'Inactiva' }}
              </AppBadge>
            </td>
            <td class="px-4 py-2.5">
              <div class="flex items-center gap-1.5">
                <AppBadge :tone="licenseTone(c.license_status)">{{ c.license_status }}</AppBadge>
                <span v-if="c.license_days_remaining !== null" class="text-xs text-gray-400">
                  ({{ c.license_days_remaining }}d)
                </span>
              </div>
            </td>
            <td class="px-4 py-2.5 text-right space-x-3">
              <button class="text-sm text-indigo-600 hover:underline" @click="openLicenseModal(c)">
                Activar licencia
              </button>
              <RouterLink :to="{ name: 'admin-client-detail', params: { id: c.id } }" class="text-sm text-indigo-600 hover:underline">
                Ver perfil
              </RouterLink>
              <button class="text-sm text-gray-500 hover:underline" @click="onToggleActive(c)">
                {{ c.is_active ? 'Desactivar' : 'Activar' }}
              </button>
            </td>
          </tr>
          <tr v-if="!loading && clients.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">Sin clientes todavía.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <AppModal :open="showCreate" title="Nuevo cliente" @close="showCreate = false">
      <form class="space-y-3" @submit.prevent="onCreate">
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Nombre completo</label>
          <input v-model="form.full_name" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Email</label>
          <input v-model="form.email" type="email" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Contraseña temporal</label>
          <input v-model="form.password" type="password" required minlength="8" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Teléfono (opcional)</label>
          <input v-model="form.phone" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
        <div class="flex justify-end gap-2 pt-2">
          <AppButton variant="secondary" type="button" @click="showCreate = false">Cancelar</AppButton>
          <AppButton type="submit">Crear cliente</AppButton>
        </div>
      </form>
    </AppModal>

    <AppModal
      :open="!!licenseTarget"
      :title="`Activar licencia — ${licenseTarget?.full_name ?? ''}`"
      @close="licenseTarget = null"
    >
      <form class="space-y-3" @submit.prevent="onActivateLicense">
        <p class="text-xs text-gray-500">
          Genera un código y lo activa de inmediato para este cliente. Si ya tiene una licencia
          vigente, la duración se sumará a su fecha de vencimiento actual.
        </p>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Duración</label>
          <select v-model="licenseForm.duration_type" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option v-for="d in durationOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
          </select>
        </div>
        <div v-if="licenseForm.duration_type === 'CUSTOM'">
          <label class="block text-xs font-medium text-gray-700 mb-1">Días</label>
          <input v-model.number="licenseForm.custom_days" type="number" min="1" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <p v-if="licenseError" class="text-sm text-red-600">{{ licenseError }}</p>
        <p v-if="licenseSuccess" class="text-sm text-green-600">{{ licenseSuccess }}</p>
        <div class="flex justify-end gap-2 pt-2">
          <AppButton variant="secondary" type="button" @click="licenseTarget = null">Cerrar</AppButton>
          <AppButton type="submit" :disabled="licenseSaving">Activar</AppButton>
        </div>
      </form>
    </AppModal>
  </div>
</template>
