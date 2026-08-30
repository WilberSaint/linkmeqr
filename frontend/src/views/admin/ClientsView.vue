<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { KeyRound, Pencil, Plus, Power, Search, Users } from '@lucide/vue'
import type { ClientWithLicense, DurationType } from '@/types'
import * as clientsApi from '@/api/clients'
import { licenseStatusLabel } from '@/composables/licenseLabels'
import AppButton from '@/components/common/AppButton.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import AppEmptyState from '@/components/common/AppEmptyState.vue'
import AppBadge from '@/components/common/AppBadge.vue'
import AppModal from '@/components/common/AppModal.vue'
import AppDropdownMenu from '@/components/common/AppDropdownMenu.vue'

const router = useRouter()

const clients = ref<ClientWithLicense[]>([])
const loading = ref(false)
const showCreate = ref(false)
const search = ref('')

const filteredClients = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return clients.value
  return clients.value.filter((c) => c.full_name.toLowerCase().includes(q) || c.email.toLowerCase().includes(q))
})

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

const editTarget = ref<ClientWithLicense | null>(null)
const editForm = ref({ full_name: '', phone: '' })
const editSaving = ref(false)
const editError = ref('')

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

function openEditModal(client: ClientWithLicense) {
  editTarget.value = client
  editError.value = ''
  editForm.value = { full_name: client.full_name, phone: client.phone ?? '' }
}

async function onSaveEdit() {
  if (!editTarget.value) return
  editError.value = ''
  editSaving.value = true
  try {
    await clientsApi.updateClient(editTarget.value.id, {
      full_name: editForm.value.full_name,
      phone: editForm.value.phone || null,
    })
    editTarget.value = null
    await load()
  } catch {
    editError.value = 'No se pudo guardar el cliente.'
  } finally {
    editSaving.value = false
  }
}

function licenseTone(status: string) {
  if (status === 'ACTIVE') return 'green'
  if (status === 'EXPIRED') return 'red'
  return 'gray'
}

function initials(name: string) {
  return name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('')
}

onMounted(load)
</script>

<template>
  <div class="p-4 sm:p-6 max-w-5xl">
    <AppPageHeader
      title="Clientes"
      description="Cada cliente es una cuenta con su propio perfil público, su QR y su licencia. Tú creas la cuenta aquí y le entregas el correo y la contraseña."
    >
      <template #actions>
        <AppButton @click="showCreate = true"><Plus :size="15" /> Nuevo cliente</AppButton>
      </template>
    </AppPageHeader>

    <div class="relative mb-4 max-w-xs">
      <Search :size="15" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
      <input
        v-model="search"
        type="text"
        placeholder="Buscar por nombre o email…"
        class="w-full rounded-lg border border-gray-300 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
      />
    </div>

    <div class="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <!-- Cliente/Cuenta/Licencia fit a phone screen; Acciones doesn't — this
           table scrolls horizontally on mobile (overflow-x-auto below), but
           nothing about that is visible until you touch-drag it, so callers
           on a phone would never discover the menu that lets them
           activate/deactivate a client without this hint. -->
      <p class="sm:hidden text-[11px] text-gray-400 px-4 pt-2.5">Desliza para ver acciones →</p>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap">Cliente</th>
              <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap">Cuenta</th>
              <th class="text-left px-4 py-2.5 font-medium whitespace-nowrap">Licencia</th>
              <th class="text-right px-4 py-2.5 font-medium whitespace-nowrap">Acciones</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr
              v-for="c in filteredClients"
              :key="c.id"
              class="cursor-pointer hover:bg-gray-50/60"
              @click="router.push({ name: 'admin-client-detail', params: { id: c.id } })"
            >
              <td class="px-4 py-2.5">
                <div class="flex items-center gap-3 min-w-[10rem]">
                  <div class="w-8 h-8 rounded-full bg-indigo-100 text-indigo-700 text-xs font-semibold flex items-center justify-center shrink-0">
                    {{ initials(c.full_name) }}
                  </div>
                  <div class="min-w-0">
                    <p class="text-gray-900 font-medium truncate">{{ c.full_name }}</p>
                    <p class="text-xs text-gray-500 truncate">{{ c.email }}</p>
                  </div>
                </div>
              </td>
              <td class="px-4 py-2.5 whitespace-nowrap">
                <AppBadge :tone="c.is_active ? 'green' : 'red'">
                  {{ c.is_active ? 'Activa' : 'Inactiva' }}
                </AppBadge>
              </td>
              <td class="px-4 py-2.5 whitespace-nowrap">
                <div class="flex items-center gap-1.5">
                  <AppBadge :tone="licenseTone(c.license_status)">{{ licenseStatusLabel(c.license_status) }}</AppBadge>
                  <span v-if="c.license_days_remaining !== null" class="text-xs text-gray-400">
                    ({{ c.license_days_remaining }}d)
                  </span>
                </div>
              </td>
              <td class="px-4 py-2.5 text-right whitespace-nowrap" @click.stop>
                <AppDropdownMenu>
                <button type="button" class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="openEditModal(c)">
                  <Pencil :size="15" /> Editar cliente
                </button>
                <button type="button" class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="openLicenseModal(c)">
                  <KeyRound :size="15" /> Activar licencia
                </button>
                <button type="button" class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="onToggleActive(c)">
                  <Power :size="15" /> {{ c.is_active ? 'Desactivar cuenta' : 'Activar cuenta' }}
                </button>
              </AppDropdownMenu>
            </td>
          </tr>
          <tr v-if="!loading && filteredClients.length === 0 && search">
            <td colspan="4" class="px-4 py-8 text-center text-gray-400">Sin resultados para tu búsqueda.</td>
          </tr>
        </tbody>
        </table>
      </div>

      <AppEmptyState
        v-if="!loading && clients.length === 0"
        :icon="Users"
        title="Todavía no tienes clientes"
        description="Crea la cuenta del negocio con su correo y una contraseña. Después podrás armarle su perfil, su QR y sus tarjetas impresas."
      >
        <template #action>
          <AppButton @click="showCreate = true"><Plus :size="15" /> Crear el primero</AppButton>
        </template>
      </AppEmptyState>
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

    <AppModal :open="!!editTarget" :title="`Editar cliente — ${editTarget?.full_name ?? ''}`" @close="editTarget = null">
      <form class="space-y-3" @submit.prevent="onSaveEdit">
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Nombre completo</label>
          <input v-model="editForm.full_name" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Teléfono</label>
          <input v-model="editForm.phone" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <p v-if="editError" class="text-sm text-red-600">{{ editError }}</p>
        <div class="flex justify-end gap-2 pt-2">
          <AppButton variant="secondary" type="button" @click="editTarget = null">Cancelar</AppButton>
          <AppButton type="submit" :disabled="editSaving">Guardar</AppButton>
        </div>
      </form>
    </AppModal>
  </div>
</template>
