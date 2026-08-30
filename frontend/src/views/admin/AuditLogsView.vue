<script setup lang="ts">
/**
 * The audit trail. Its whole job is answering "who did what, when" — so the
 * actor is now a column rather than an id the API returned and the screen
 * dropped on the floor, and the list can be narrowed instead of only ever
 * showing the last 500 rows of everything.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { ScrollText } from '@lucide/vue'

import { apiClient } from '@/api/client'
import { auditActionLabel, auditEntityLabel } from '@/composables/auditLabels'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import AppEmptyState from '@/components/common/AppEmptyState.vue'

interface AuditLog {
  id: string
  actor_user_id: string | null
  actor_name: string | null
  actor_email: string | null
  action: string
  entity_type: string
  entity_id: string | null
  created_at: string
}

const logs = ref<AuditLog[]>([])
const loading = ref(true)
const entityFilter = ref('')
const actionFilter = ref('')

async function load() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (entityFilter.value) params.entity_type = entityFilter.value
    if (actionFilter.value) params.action = actionFilter.value
    logs.value = await apiClient.get<AuditLog[]>('/admin/audit-logs', { params }).then((r) => r.data)
  } finally {
    loading.value = false
  }
}

/**
 * Filter options come from the rows themselves rather than a hardcoded list,
 * so a newly added action shows up here without anyone remembering to add it.
 * Loaded unfiltered once, so narrowing the list never shrinks the choices.
 */
const allEntities = ref<string[]>([])
const allActions = ref<string[]>([])

async function loadFilterOptions() {
  const rows = await apiClient.get<AuditLog[]>('/admin/audit-logs').then((r) => r.data)
  allEntities.value = [...new Set(rows.map((l) => l.entity_type))].sort()
  allActions.value = [...new Set(rows.map((l) => l.action))].sort()
}

watch([entityFilter, actionFilter], load)

const isFiltered = computed(() => Boolean(entityFilter.value || actionFilter.value))

function clearFilters() {
  entityFilter.value = ''
  actionFilter.value = ''
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('es-MX', { dateStyle: 'medium', timeStyle: 'short' })
}

function shortId(id: string | null) {
  if (!id) return null
  return id.length > 8 ? `${id.slice(0, 8)}…` : id
}

/** Initials stand in for an avatar, so scanning the column groups by person at a glance. */
function initials(name: string | null) {
  if (!name) return '—'
  return name
    .split(' ')
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

onMounted(() => {
  load()
  loadFilterOptions()
})
</script>

<template>
  <div class="max-w-5xl p-4 sm:p-6">
    <AppPageHeader
      title="Registro de auditoría"
      description="Todo lo que se ha hecho en el panel: quién lo hizo, sobre qué y cuándo. Se guarda automáticamente."
    />

    <div class="mb-4 flex flex-wrap items-end gap-3">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Acción</span>
        <select v-model="actionFilter" class="min-w-48 rounded-lg border border-gray-300 px-3 py-2 text-sm">
          <option value="">Todas</option>
          <option v-for="a in allActions" :key="a" :value="a">{{ auditActionLabel(a) }}</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Entidad</span>
        <select v-model="entityFilter" class="min-w-40 rounded-lg border border-gray-300 px-3 py-2 text-sm">
          <option value="">Todas</option>
          <option v-for="e in allEntities" :key="e" :value="e">{{ auditEntityLabel(e) }}</option>
        </select>
      </label>
      <button
        v-if="isFiltered"
        type="button"
        class="pb-2 text-sm text-gray-500 hover:text-gray-700 hover:underline"
        @click="clearFilters"
      >
        Limpiar filtros
      </button>
      <span class="pb-2 text-xs text-gray-400">{{ logs.length }} registros</span>
    </div>

    <div class="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 text-xs uppercase text-gray-500">
            <tr>
              <th class="whitespace-nowrap px-4 py-3 text-left font-medium">Quién</th>
              <th class="whitespace-nowrap px-4 py-3 text-left font-medium">Acción</th>
              <th class="whitespace-nowrap px-4 py-3 text-left font-medium">Entidad</th>
              <th class="whitespace-nowrap px-4 py-3 text-left font-medium">Cuándo</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-for="log in logs" :key="log.id" class="hover:bg-gray-50">
              <td class="whitespace-nowrap px-4 py-2.5">
                <div class="flex items-center gap-2">
                  <span
                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-50 text-[10px] font-semibold text-indigo-600"
                  >
                    {{ initials(log.actor_name) }}
                  </span>
                  <span class="min-w-0">
                    <span class="block truncate text-gray-900">{{ log.actor_name ?? 'Sistema' }}</span>
                    <span v-if="log.actor_email" class="block truncate text-xs text-gray-400">{{ log.actor_email }}</span>
                  </span>
                </div>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 text-gray-900">{{ auditActionLabel(log.action) }}</td>
              <td class="whitespace-nowrap px-4 py-2.5 text-gray-600">
                {{ auditEntityLabel(log.entity_type) }}
                <span v-if="shortId(log.entity_id)" class="ml-1 font-mono text-xs text-gray-400">
                  #{{ shortId(log.entity_id) }}
                </span>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 text-gray-500">{{ formatDate(log.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <AppEmptyState
        v-if="!loading && logs.length === 0"
        :icon="ScrollText"
        :title="isFiltered ? 'Ningún registro coincide' : 'Sin actividad todavía'"
        :description="
          isFiltered
            ? 'Prueba quitando algún filtro.'
            : 'Aquí aparecerá cada acción del panel en cuanto empieces a crear clientes, códigos o tarjetas.'
        "
      />
      <p v-if="loading" class="px-4 py-10 text-center text-sm text-gray-400">Cargando…</p>
    </div>
  </div>
</template>
