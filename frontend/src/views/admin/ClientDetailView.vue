<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { LicenseActivation, Profile, User } from '@/types'
import { apiClient } from '@/api/client'
import * as clientsApi from '@/api/clients'
import * as licensesApi from '@/api/licenses'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'

const route = useRoute()
const clientId = route.params.id as string

const client = ref<User | null>(null)
const profile = ref<Profile | null>(null)
const history = ref<LicenseActivation[]>([])
const loading = ref(true)

const profileForm = ref({ business_name: '', slug: '' })
const creatingProfile = ref(false)
const createError = ref('')

async function load() {
  loading.value = true
  try {
    client.value = await clientsApi.getClient(clientId)
    history.value = await licensesApi.clientLicenseHistory(clientId)
    try {
      profile.value = await apiClient.get<Profile>(`/admin/clients/${clientId}/profile`).then((r) => r.data)
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
    profile.value = await apiClient
      .post<Profile>(`/admin/clients/${clientId}/profile`, {
        business_name: profileForm.value.business_name,
        slug: profileForm.value.slug,
      })
      .then((r) => r.data)
  } catch {
    createError.value = 'No se pudo crear el perfil (¿slug ya en uso?).'
  } finally {
    creatingProfile.value = false
  }
}

function formatDate(d: string | null) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('es-MX', { year: 'numeric', month: 'short', day: 'numeric' })
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-3xl space-y-6">
    <div>
      <RouterLink :to="{ name: 'admin-clients' }" class="text-sm text-indigo-600 hover:underline">← Clientes</RouterLink>
    </div>

    <div v-if="client">
      <h1 class="text-lg font-semibold text-gray-900">{{ client.full_name }}</h1>
      <p class="text-sm text-gray-500">{{ client.email }}</p>
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
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Historial de licencia</h2>
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
          <tr v-if="!loading && history.length === 0">
            <td colspan="4" class="py-6 text-center text-gray-400">Sin activaciones todavía.</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
