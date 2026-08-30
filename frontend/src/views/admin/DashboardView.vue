<script setup lang="ts">
/**
 * The admin home.
 *
 * It used to spend half its space on cards linking to Clientes, Licencias and
 * Plantillas — the same three items the sidebar already shows, one click away.
 * That space now goes to the things that actually need someone to act: licences
 * about to lapse, clients who never got a public profile, and accounts that are
 * switched off.
 */
import { computed, onMounted, ref } from 'vue'
import { Users, ShieldCheck, Ticket, Globe, AlertTriangle, ArrowRight } from '@lucide/vue'

import type { ActivationCode, ClientWithLicense, Profile } from '@/types'
import * as clientsApi from '@/api/clients'
import * as licensesApi from '@/api/licenses'
import { apiClient } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import AppStatCard from '@/components/common/AppStatCard.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import { auditActionLabel } from '@/composables/auditLabels'

interface AuditLogRow {
  id: string
  action: string
  entity_type: string
  actor_name: string | null
  created_at: string
}

const auth = useAuthStore()
const loading = ref(true)
const clients = ref<ClientWithLicense[]>([])
const codes = ref<ActivationCode[]>([])
const profiles = ref<Profile[]>([])
const recentLogs = ref<AuditLogRow[]>([])

const activeLicenses = computed(() => clients.value.filter((c) => c.license_status === 'ACTIVE').length)
const unusedCodes = computed(() => codes.value.filter((c) => c.status === 'UNUSED').length)
const publishedProfiles = computed(() => profiles.value.filter((p) => p.is_published).length)

/** Two weeks is enough notice to reach a client before their card stops working. */
const EXPIRY_WARNING_DAYS = 15

const expiringSoon = computed(() =>
  clients.value
    .filter(
      (c) =>
        c.license_status === 'ACTIVE' &&
        c.license_days_remaining !== null &&
        c.license_days_remaining <= EXPIRY_WARNING_DAYS,
    )
    .sort((a, b) => (a.license_days_remaining ?? 0) - (b.license_days_remaining ?? 0)),
)

const expired = computed(() => clients.value.filter((c) => c.license_status === 'EXPIRED'))

/**
 * A client with no profile has nothing for their QR to point at, so every card
 * printed for them leads nowhere — worth surfacing before it gets printed.
 */
const withoutProfile = computed(() => {
  const owners = new Set(profiles.value.map((p) => p.user_id))
  return clients.value.filter((c) => !owners.has(c.id))
})

const needsAttention = computed(
  () => expiringSoon.value.length + expired.value.length + withoutProfile.value.length,
)

function formatDate(d: string) {
  return new Date(d).toLocaleString('es-MX', { dateStyle: 'medium', timeStyle: 'short' })
}

async function load() {
  loading.value = true
  try {
    const [clientsRes, codesRes, profilesRes, logsRes] = await Promise.all([
      clientsApi.listClients(),
      licensesApi.listCodes(),
      apiClient.get<Profile[]>('/admin/profiles').then((r) => r.data),
      apiClient.get<AuditLogRow[]>('/admin/audit-logs', { params: { limit: 8 } }).then((r) => r.data),
    ])
    clients.value = clientsRes
    codes.value = codesRes
    profiles.value = profilesRes
    recentLogs.value = logsRes
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="max-w-5xl space-y-6 p-4 sm:p-6">
    <AppPageHeader title="Panel administrativo" :description="`Hola, ${auth.user?.full_name ?? ''}.`" />

    <div class="grid grid-cols-2 gap-4 md:grid-cols-4">
      <RouterLink :to="{ name: 'admin-clients' }">
        <AppStatCard label="Clientes" :value="loading ? '—' : clients.length" :icon="Users" tone="indigo" />
      </RouterLink>
      <AppStatCard label="Licencias activas" :value="loading ? '—' : activeLicenses" :icon="ShieldCheck" tone="green" />
      <RouterLink :to="{ name: 'admin-licenses' }">
        <AppStatCard label="Códigos sin usar" :value="loading ? '—' : unusedCodes" :icon="Ticket" tone="amber" />
      </RouterLink>
      <AppStatCard label="Perfiles publicados" :value="loading ? '—' : publishedProfiles" :icon="Globe" tone="gray" />
    </div>

    <!-- Needs attention -->
    <section class="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div class="flex items-center gap-2 border-b border-gray-100 px-5 py-3.5">
        <AlertTriangle :size="15" :class="needsAttention > 0 ? 'text-amber-500' : 'text-gray-300'" />
        <h2 class="text-sm font-semibold text-gray-900">Necesita atención</h2>
        <span v-if="needsAttention > 0" class="ml-auto text-xs text-gray-400">{{ needsAttention }}</span>
      </div>

      <p v-if="!loading && needsAttention === 0" class="px-5 py-8 text-center text-sm text-gray-400">
        Todo en orden. Ninguna licencia por vencer y todos los clientes tienen perfil.
      </p>

      <ul v-else class="divide-y divide-gray-100">
        <li v-for="c in expired" :key="`exp-${c.id}`">
          <RouterLink
            :to="{ name: 'admin-client-detail', params: { id: c.id } }"
            class="flex items-center gap-3 px-5 py-3 text-sm hover:bg-gray-50"
          >
            <span class="h-2 w-2 shrink-0 rounded-full bg-red-500" />
            <span class="min-w-0 flex-1 truncate text-gray-900">{{ c.full_name }}</span>
            <span class="shrink-0 text-xs text-red-600">Licencia vencida</span>
            <ArrowRight :size="14" class="shrink-0 text-gray-300" />
          </RouterLink>
        </li>
        <li v-for="c in expiringSoon" :key="`soon-${c.id}`">
          <RouterLink
            :to="{ name: 'admin-client-detail', params: { id: c.id } }"
            class="flex items-center gap-3 px-5 py-3 text-sm hover:bg-gray-50"
          >
            <span class="h-2 w-2 shrink-0 rounded-full bg-amber-500" />
            <span class="min-w-0 flex-1 truncate text-gray-900">{{ c.full_name }}</span>
            <span class="shrink-0 text-xs text-amber-600">
              Vence en {{ c.license_days_remaining }} {{ c.license_days_remaining === 1 ? 'día' : 'días' }}
            </span>
            <ArrowRight :size="14" class="shrink-0 text-gray-300" />
          </RouterLink>
        </li>
        <li v-for="c in withoutProfile" :key="`np-${c.id}`">
          <RouterLink
            :to="{ name: 'admin-client-detail', params: { id: c.id } }"
            class="flex items-center gap-3 px-5 py-3 text-sm hover:bg-gray-50"
          >
            <span class="h-2 w-2 shrink-0 rounded-full bg-gray-300" />
            <span class="min-w-0 flex-1 truncate text-gray-900">{{ c.full_name }}</span>
            <span class="shrink-0 text-xs text-gray-500">Sin perfil público</span>
            <ArrowRight :size="14" class="shrink-0 text-gray-300" />
          </RouterLink>
        </li>
      </ul>
    </section>

    <!-- Recent activity -->
    <section class="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div class="flex items-center justify-between border-b border-gray-100 px-5 py-3.5">
        <h2 class="text-sm font-semibold text-gray-900">Actividad reciente</h2>
        <RouterLink :to="{ name: 'admin-audit' }" class="text-xs text-indigo-600 hover:underline">Ver todo</RouterLink>
      </div>
      <ul class="divide-y divide-gray-100">
        <li v-for="log in recentLogs" :key="log.id" class="flex items-center gap-3 px-5 py-2.5 text-sm">
          <span class="min-w-0 flex-1 truncate text-gray-700">{{ auditActionLabel(log.action) }}</span>
          <span v-if="log.actor_name" class="shrink-0 text-xs text-gray-500">{{ log.actor_name }}</span>
          <span class="shrink-0 text-xs text-gray-400">{{ formatDate(log.created_at) }}</span>
        </li>
        <li v-if="!loading && recentLogs.length === 0" class="px-5 py-8 text-center text-sm text-gray-400">
          Sin actividad todavía.
        </li>
      </ul>
    </section>
  </div>
</template>
