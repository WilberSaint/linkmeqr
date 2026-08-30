<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Eye, MousePointerClick, TrendingUp, Calendar, Copy, Check, ExternalLink } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { useLicenseStore } from '@/stores/license'
import * as statsApi from '@/api/stats'
import type { StatsSummary } from '@/api/stats'
import { licenseStatusLabel } from '@/composables/licenseLabels'
import { blockLabel } from '@/composables/blockLabels'
import type { BlockType } from '@/types'
import AppBadge from '@/components/common/AppBadge.vue'
import AppStatCard from '@/components/common/AppStatCard.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import * as profileApi from '@/api/profile'

const auth = useAuthStore()
const licenseStore = useLicenseStore()
const range = ref<'7d' | '30d'>('30d')
const summary = ref<StatsSummary | null>(null)
const statsLoading = ref(true)

const maxBlockClicks = computed(() => Math.max(1, ...(summary.value?.block_clicks.map((b) => b.count) ?? [1])))

// The API only returns rows for days that actually had a view — with a
// fresh profile that's often just 1-2 days, which made the bar chart
// render as a single block filling the whole width instead of a proper
// 7/30-day chart. Pad the missing days with 0 so the x-axis always spans
// the selected range.
const paddedTimeseries = computed(() => {
  const days = range.value === '7d' ? 7 : 30
  const counts = new Map((summary.value?.timeseries ?? []).map((d) => [d.date.slice(0, 10), d.count]))
  const today = new Date()
  const out: { date: string; count: number }[] = []
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    const iso = d.toISOString().slice(0, 10)
    out.push({ date: iso, count: counts.get(iso) ?? 0 })
  }
  return out
})

const maxDailyViews = computed(() => Math.max(1, ...paddedTimeseries.value.map((d) => d.count)))

function blockRowLabel(title: string, type: string) {
  return title || blockLabel(type as BlockType)
}

function statusTone(status?: string) {
  if (status === 'ACTIVE') return 'green'
  if (status === 'EXPIRED') return 'red'
  return 'gray'
}

/**
 * The client's own public link. It is the thing they are actually here to
 * hand out — on a card, in a bio, over WhatsApp — so it belongs on the first
 * screen rather than buried in the profile editor.
 */
const slug = ref('')
const copied = ref(false)

const publicUrl = computed(() => (slug.value ? `${window.location.origin}/p/${slug.value}` : ''))

async function copyLink() {
  if (!publicUrl.value) return
  await navigator.clipboard.writeText(publicUrl.value)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

/** Below this the expiry stops being a date and starts being a problem. */
const EXPIRY_WARNING_DAYS = 15
const daysLeft = computed(() => licenseStore.license?.days_remaining ?? null)
const isExpiring = computed(
  () => licenseStore.license?.status === 'ACTIVE' && daysLeft.value !== null && daysLeft.value <= EXPIRY_WARNING_DAYS,
)

async function load() {
  statsLoading.value = true
  try {
    summary.value = await statsApi.myStatsSummary(range.value)
  } finally {
    statsLoading.value = false
  }
  try {
    slug.value = (await profileApi.getMyProfile()).slug
  } catch {
    slug.value = ''
  }
}

watch(range, load)
onMounted(async () => {
  licenseStore.refresh()
  await load()
})
</script>

<template>
  <div class="p-6 max-w-4xl space-y-6">
    <AppPageHeader :title="`Hola, ${auth.user?.full_name ?? ''}`" description="Así va tu perfil público.">
      <template #actions>
        <select v-model="range" class="rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
          <option value="7d">Últimos 7 días</option>
          <option value="30d">Últimos 30 días</option>
        </select>
      </template>
    </AppPageHeader>

    <!-- The link, front and centre: it is what the whole product produces. -->
    <section v-if="publicUrl" class="rounded-xl border border-gray-200 bg-white p-5">
      <p class="text-xs font-medium uppercase tracking-wide text-gray-400">Tu enlace público</p>
      <div class="mt-2 flex flex-wrap items-center gap-2">
        <code class="min-w-0 flex-1 truncate rounded-lg bg-gray-50 px-3 py-2 font-mono text-sm text-gray-800">
          {{ publicUrl }}
        </code>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
          @click="copyLink"
        >
          <component :is="copied ? Check : Copy" :size="14" />
          {{ copied ? 'Copiado' : 'Copiar' }}
        </button>
        <a
          :href="publicUrl"
          target="_blank"
          rel="noopener"
          class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
        >
          <ExternalLink :size="14" /> Abrir
        </a>
      </div>
    </section>

    <RouterLink
      :to="{ name: 'client-license' }"
      class="flex items-center justify-between rounded-xl border p-5 transition"
      :class="
        licenseStore.license?.status === 'ACTIVE'
          ? isExpiring
            ? 'border-amber-200 bg-amber-50 hover:border-amber-300'
            : 'border-gray-200 bg-white hover:border-gray-300'
          : 'border-red-200 bg-red-50 hover:border-red-300'
      "
    >
      <div>
        <p class="text-sm text-gray-500">Estado de tu licencia</p>
        <p v-if="daysLeft !== null" class="mt-1 text-sm font-medium text-gray-900">
          {{ daysLeft }} {{ daysLeft === 1 ? 'día restante' : 'días restantes' }}
        </p>
        <p v-if="isExpiring" class="mt-1 text-xs text-amber-700">Está por vencer — toca para renovar.</p>
        <p v-else-if="licenseStore.license?.status !== 'ACTIVE'" class="mt-1 text-xs text-red-700">
          Tu perfil no se está mostrando. Toca para activar un código.
        </p>
      </div>
      <AppBadge :tone="statusTone(licenseStore.license?.status)">
        {{ licenseStatusLabel(licenseStore.license?.status) }}
      </AppBadge>
    </RouterLink>

    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <AppStatCard label="Visitas totales" :value="statsLoading ? '—' : (summary?.total_views ?? 0)" :icon="Eye" tone="indigo" />
      <AppStatCard label="Clics en enlaces" :value="statsLoading ? '—' : (summary?.total_clicks ?? 0)" :icon="MousePointerClick" tone="indigo" />
      <AppStatCard label="Últimos 7 días" :value="statsLoading ? '—' : (summary?.views_7d ?? 0)" :icon="TrendingUp" tone="green" />
      <AppStatCard label="Últimos 30 días" :value="statsLoading ? '—' : (summary?.views_30d ?? 0)" :icon="Calendar" tone="gray" />
    </div>

    <div class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Visitas por día</h2>
      <div class="flex items-end gap-1 h-32">
        <div
          v-for="d in paddedTimeseries"
          :key="d.date"
          class="flex-1 bg-indigo-500 rounded-t min-w-0"
          :class="{ 'bg-gray-100': d.count === 0 }"
          :style="{ height: `${d.count === 0 ? 2 : Math.max(4, (d.count / maxDailyViews) * 100)}%` }"
          :title="`${d.date}: ${d.count}`"
        />
      </div>
    </div>

    <div class="bg-white border border-gray-200 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Clics por bloque</h2>
      <div v-if="summary?.block_clicks.length" class="space-y-2.5">
        <div v-for="row in summary.block_clicks" :key="row.block_id">
          <div class="flex items-center justify-between text-xs mb-1">
            <span class="text-gray-700 truncate pr-2">{{ blockRowLabel(row.title, row.block_type) }}</span>
            <span class="text-gray-400 shrink-0 tabular-nums">{{ row.count }}</span>
          </div>
          <div class="h-2.5 rounded-full bg-gray-100 overflow-hidden">
            <div class="h-full bg-indigo-500 rounded-full" :style="{ width: `${Math.max(4, (row.count / maxBlockClicks) * 100)}%` }" />
          </div>
        </div>
      </div>
      <p v-else class="text-sm text-gray-400 text-center py-4">Sin clics todavía.</p>
    </div>

    <div v-if="summary" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="bg-white border border-gray-200 rounded-xl p-4">
        <h3 class="text-xs font-semibold text-gray-500 mb-2">Dispositivos</h3>
        <div v-for="d in summary.devices" :key="d.label" class="flex justify-between text-sm py-0.5">
          <span class="text-gray-600">{{ d.label }}</span>
          <span class="text-gray-900 font-medium">{{ d.count }}</span>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-xl p-4">
        <h3 class="text-xs font-semibold text-gray-500 mb-2">Sistema operativo</h3>
        <div v-for="d in summary.os" :key="d.label" class="flex justify-between text-sm py-0.5">
          <span class="text-gray-600">{{ d.label }}</span>
          <span class="text-gray-900 font-medium">{{ d.count }}</span>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-xl p-4">
        <h3 class="text-xs font-semibold text-gray-500 mb-2">Navegador</h3>
        <div v-for="d in summary.browsers" :key="d.label" class="flex justify-between text-sm py-0.5">
          <span class="text-gray-600">{{ d.label }}</span>
          <span class="text-gray-900 font-medium">{{ d.count }}</span>
        </div>
      </div>
    </div>

  </div>
</template>
