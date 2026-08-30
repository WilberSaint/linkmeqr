<script setup lang="ts">
/**
 * Activation codes.
 *
 * The screen now says out loud how a client actually gets a licence, because
 * the two routes are not obvious from a list of codes: either you activate one
 * directly from the client's own row, or you generate a code and the client
 * redeems it. And a code is a bearer token unless you reserve it for someone —
 * that distinction was modelled in the database and shown in the table, but
 * there was no way to actually set it.
 */
import { computed, onMounted, ref } from 'vue'
import { Ticket, Copy, Check } from '@lucide/vue'

import type { ActivationCode, ClientWithLicense, DurationType } from '@/types'
import * as licensesApi from '@/api/licenses'
import * as clientsApi from '@/api/clients'
import { codeStatusLabel } from '@/composables/licenseLabels'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import AppEmptyState from '@/components/common/AppEmptyState.vue'

const codes = ref<ActivationCode[]>([])
const clients = ref<ClientWithLicense[]>([])
const loading = ref(false)
const statusFilter = ref('')

const durationOptions: { value: DurationType; label: string }[] = [
  { value: '1_MONTH', label: '1 mes' },
  { value: '3_MONTHS', label: '3 meses' },
  { value: '6_MONTHS', label: '6 meses' },
  { value: '1_YEAR', label: '1 año' },
  { value: 'CUSTOM', label: 'Personalizado' },
]

const form = ref<{ duration_type: DurationType; custom_days: number; quantity: number; assign_to: string }>({
  duration_type: '1_MONTH',
  custom_days: 30,
  quantity: 1,
  assign_to: '',
})

const lastGenerated = ref<ActivationCode[]>([])
const generating = ref(false)
const error = ref('')
const copied = ref<string | null>(null)

async function load() {
  loading.value = true
  try {
    codes.value = await licensesApi.listCodes(statusFilter.value ? { status: statusFilter.value } : undefined)
  } finally {
    loading.value = false
  }
}

async function loadClients() {
  try {
    clients.value = await clientsApi.listClients()
  } catch {
    clients.value = []
  }
}

/**
 * One form for both cases: a quantity of 1 is a single code, more is a batch.
 * Two near-identical side-by-side forms only made the reader compare them to
 * find the difference.
 */
async function onGenerate() {
  error.value = ''
  generating.value = true
  try {
    const payload = { duration_type: form.value.duration_type, custom_days: form.value.custom_days }
    const result =
      form.value.quantity > 1
        ? await licensesApi.generateBatch({ ...payload, quantity: form.value.quantity })
        : [await licensesApi.generateCode(payload)]

    // Reserving on creation is the common case for a code made for one client;
    // a batch is left unassigned, since its whole point is being handed out.
    if (form.value.assign_to && result.length === 1) {
      await licensesApi.assignCode(result[0].id, form.value.assign_to)
    }
    lastGenerated.value = result
    await load()
  } catch {
    error.value = 'No se pudo generar el código.'
  } finally {
    generating.value = false
  }
}

async function onAssign(code: ActivationCode, userId: string) {
  try {
    await licensesApi.assignCode(code.id, userId || null)
    await load()
  } catch {
    error.value = 'No se pudo reservar el código.'
  }
}

async function onRevoke(id: string) {
  if (!confirm('¿Revocar este código? Dejará de poder canjearse.')) return
  await licensesApi.revokeCode(id)
  await load()
}

async function copy(text: string, key: string) {
  await navigator.clipboard.writeText(text)
  copied.value = key
  setTimeout(() => (copied.value = null), 1500)
}

function copyAll() {
  copy(lastGenerated.value.map((c) => c.code).join('\n'), 'all')
}

function statusTone(status: string) {
  if (status === 'UNUSED') return 'amber'
  if (status === 'USED') return 'green'
  return 'gray'
}

const unusedCount = computed(() => codes.value.filter((c) => c.status === 'UNUSED').length)

onMounted(() => {
  load()
  loadClients()
})
</script>

<template>
  <div class="max-w-5xl space-y-6 p-4 sm:p-6">
    <AppPageHeader
      title="Licencias y códigos"
      description="Un código de activación es un vale por cierto tiempo de licencia. El cliente lo canjea desde su panel y su licencia se extiende."
    />

    <!-- Generate -->
    <section class="rounded-xl border border-gray-200 bg-white p-5">
      <h2 class="mb-4 text-sm font-semibold text-gray-900">Generar códigos</h2>
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="onGenerate">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Duración</span>
          <select v-model="form.duration_type" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option v-for="d in durationOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
          </select>
        </label>

        <label v-if="form.duration_type === 'CUSTOM'" class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Días</span>
          <input
            v-model.number="form.custom_days"
            type="number"
            min="1"
            class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>

        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Cantidad</span>
          <input
            v-model.number="form.quantity"
            type="number"
            min="1"
            max="5000"
            class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>

        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Reservar para</span>
          <select
            v-model="form.assign_to"
            :disabled="form.quantity > 1"
            class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-400"
          >
            <option value="">Cualquiera (al portador)</option>
            <option v-for="c in clients" :key="c.id" :value="c.id">{{ c.full_name }}</option>
          </select>
          <span v-if="form.quantity > 1" class="mt-1 block text-xs text-gray-400">
            Un lote se reparte, así que va sin reservar.
          </span>
        </label>

        <div class="flex items-end sm:col-span-4">
          <AppButton type="submit" :disabled="generating">
            {{ generating ? 'Generando…' : form.quantity > 1 ? `Generar ${form.quantity} códigos` : 'Generar código' }}
          </AppButton>
        </div>
      </form>
    </section>

    <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

    <section v-if="lastGenerated.length" class="rounded-xl border border-green-200 bg-green-50 p-5">
      <div class="mb-3 flex items-center justify-between gap-3">
        <h2 class="text-sm font-semibold text-green-900">
          Listo — {{ lastGenerated.length }} {{ lastGenerated.length === 1 ? 'código generado' : 'códigos generados' }}
        </h2>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border border-green-300 bg-white px-3 py-1.5 text-xs font-medium text-green-800 hover:bg-green-100"
          @click="copyAll"
        >
          <component :is="copied === 'all' ? Check : Copy" :size="13" />
          {{ copied === 'all' ? 'Copiado' : 'Copiar todos' }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="c in lastGenerated"
          :key="c.id"
          type="button"
          class="rounded border border-green-200 bg-white px-2.5 py-1.5 font-mono text-xs text-green-900 hover:border-green-400"
          :title="'Copiar ' + c.code"
          @click="copy(c.code, c.id)"
        >
          {{ copied === c.id ? '¡Copiado!' : c.code }}
        </button>
      </div>
    </section>

    <!-- All codes -->
    <section class="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-4 py-3">
        <h2 class="text-sm font-semibold text-gray-900">
          Todos los códigos
          <span class="ml-1 font-normal text-gray-400">· {{ unusedCount }} sin usar</span>
        </h2>
        <select
          v-model="statusFilter"
          class="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs"
          @change="load"
        >
          <option value="">Todos</option>
          <option value="UNUSED">Sin usar</option>
          <option value="USED">Usados</option>
          <option value="REVOKED">Revocados</option>
        </select>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 text-xs uppercase text-gray-500">
            <tr>
              <th class="whitespace-nowrap px-4 py-2.5 text-left font-medium">Código</th>
              <th class="whitespace-nowrap px-4 py-2.5 text-left font-medium">Duración</th>
              <th class="whitespace-nowrap px-4 py-2.5 text-left font-medium">Estado</th>
              <th class="whitespace-nowrap px-4 py-2.5 text-left font-medium">Cliente</th>
              <th class="whitespace-nowrap px-4 py-2.5 text-right font-medium">Acciones</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-for="c in codes" :key="c.id" class="hover:bg-gray-50">
              <td class="whitespace-nowrap px-4 py-2.5">
                <button
                  type="button"
                  class="font-mono text-xs text-gray-900 hover:text-indigo-600"
                  :title="'Copiar ' + c.code"
                  @click="copy(c.code, c.id)"
                >
                  {{ copied === c.id ? '¡Copiado!' : c.code }}
                </button>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 text-gray-600">{{ c.duration_days }} días</td>
              <td class="whitespace-nowrap px-4 py-2.5">
                <AppBadge :tone="statusTone(c.status)">{{ codeStatusLabel(c.status) }}</AppBadge>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5">
                <!-- Once redeemed the owner is a fact; before that it is a choice. -->
                <RouterLink
                  v-if="c.used_by_user_id && c.used_by_name"
                  :to="{ name: 'admin-client-detail', params: { id: c.used_by_user_id } }"
                  class="text-indigo-600 hover:underline"
                >
                  {{ c.used_by_name }}
                </RouterLink>
                <select
                  v-else-if="c.status === 'UNUSED'"
                  :value="c.assigned_user_id ?? ''"
                  class="rounded-lg border border-gray-200 px-2 py-1 text-xs"
                  @change="onAssign(c, ($event.target as HTMLSelectElement).value)"
                >
                  <option value="">Al portador</option>
                  <option v-for="cl in clients" :key="cl.id" :value="cl.id">{{ cl.full_name }}</option>
                </select>
                <span v-else class="text-gray-400">—</span>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5 text-right">
                <AppButton v-if="c.status === 'UNUSED'" variant="ghost" @click="onRevoke(c.id)">Revocar</AppButton>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AppEmptyState
        v-if="!loading && codes.length === 0"
        :icon="Ticket"
        title="Sin códigos todavía"
        description="Genera uno arriba y pásaselo a tu cliente, o activa su licencia directamente desde la pantalla de Clientes."
      />
    </section>
  </div>
</template>
