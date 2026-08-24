<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { ActivationCode, DurationType } from '@/types'
import * as licensesApi from '@/api/licenses'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'

const codes = ref<ActivationCode[]>([])
const loading = ref(false)
const statusFilter = ref('')

const durationOptions: { value: DurationType; label: string }[] = [
  { value: '1_MONTH', label: '1 mes' },
  { value: '3_MONTHS', label: '3 meses' },
  { value: '6_MONTHS', label: '6 meses' },
  { value: '1_YEAR', label: '1 año' },
  { value: 'CUSTOM', label: 'Personalizado' },
]

const singleForm = ref<{ duration_type: DurationType; custom_days: number }>({
  duration_type: '1_MONTH',
  custom_days: 30,
})
const batchForm = ref<{ duration_type: DurationType; custom_days: number; quantity: number }>({
  duration_type: '1_MONTH',
  custom_days: 30,
  quantity: 10,
})

const lastGenerated = ref<ActivationCode[]>([])
const generating = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  try {
    codes.value = await licensesApi.listCodes(statusFilter.value ? { status: statusFilter.value } : undefined)
  } finally {
    loading.value = false
  }
}

async function onGenerateSingle() {
  error.value = ''
  generating.value = true
  try {
    const code = await licensesApi.generateCode(singleForm.value)
    lastGenerated.value = [code]
    await load()
  } catch {
    error.value = 'No se pudo generar el código.'
  } finally {
    generating.value = false
  }
}

async function onGenerateBatch() {
  error.value = ''
  generating.value = true
  try {
    const codesResult = await licensesApi.generateBatch(batchForm.value)
    lastGenerated.value = codesResult
    await load()
  } catch {
    error.value = 'No se pudo generar el lote.'
  } finally {
    generating.value = false
  }
}

async function onRevoke(id: string) {
  await licensesApi.revokeCode(id)
  await load()
}

function statusTone(status: string) {
  if (status === 'UNUSED') return 'amber'
  if (status === 'USED') return 'green'
  return 'gray'
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-5xl space-y-6">
    <h1 class="text-lg font-semibold text-gray-900">Licencias y códigos de activación</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <section class="bg-white border border-gray-200 rounded-xl p-5">
        <h2 class="text-sm font-semibold text-gray-900 mb-3">Generar código individual</h2>
        <form class="space-y-3" @submit.prevent="onGenerateSingle">
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Duración</label>
            <select v-model="singleForm.duration_type" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
              <option v-for="d in durationOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
            </select>
          </div>
          <div v-if="singleForm.duration_type === 'CUSTOM'">
            <label class="block text-xs font-medium text-gray-700 mb-1">Días</label>
            <input v-model.number="singleForm.custom_days" type="number" min="1" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <AppButton type="submit" :disabled="generating">Generar código</AppButton>
        </form>
      </section>

      <section class="bg-white border border-gray-200 rounded-xl p-5">
        <h2 class="text-sm font-semibold text-gray-900 mb-3">Generar por lote</h2>
        <form class="space-y-3" @submit.prevent="onGenerateBatch">
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Duración</label>
            <select v-model="batchForm.duration_type" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
              <option v-for="d in durationOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
            </select>
          </div>
          <div v-if="batchForm.duration_type === 'CUSTOM'">
            <label class="block text-xs font-medium text-gray-700 mb-1">Días</label>
            <input v-model.number="batchForm.custom_days" type="number" min="1" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Cantidad</label>
            <input v-model.number="batchForm.quantity" type="number" min="1" max="5000" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <AppButton type="submit" :disabled="generating">Generar lote</AppButton>
        </form>
      </section>
    </div>

    <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

    <section v-if="lastGenerated.length" class="bg-indigo-50 border border-indigo-100 rounded-xl p-5">
      <h2 class="text-sm font-semibold text-indigo-900 mb-2">
        Códigos generados ({{ lastGenerated.length }})
      </h2>
      <div class="flex flex-wrap gap-2">
        <code
          v-for="c in lastGenerated"
          :key="c.id"
          class="bg-white border border-indigo-200 rounded px-2 py-1 text-xs text-indigo-900"
        >
          {{ c.code }}
        </code>
      </div>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <div class="flex items-center justify-between px-4 py-3 border-b border-gray-100">
        <h2 class="text-sm font-semibold text-gray-900">Todos los códigos</h2>
        <select v-model="statusFilter" class="rounded-lg border border-gray-300 px-2 py-1 text-xs" @change="load">
          <option value="">Todos</option>
          <option value="UNUSED">Sin usar</option>
          <option value="USED">Usados</option>
          <option value="REVOKED">Revocados</option>
        </select>
      </div>
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
          <tr>
            <th class="text-left px-4 py-2 font-medium">Código</th>
            <th class="text-left px-4 py-2 font-medium">Duración</th>
            <th class="text-left px-4 py-2 font-medium">Estado</th>
            <th class="text-right px-4 py-2 font-medium">Acciones</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="c in codes" :key="c.id">
            <td class="px-4 py-2"><code class="text-xs">{{ c.code }}</code></td>
            <td class="px-4 py-2 text-gray-600">{{ c.duration_days }} días</td>
            <td class="px-4 py-2"><AppBadge :tone="statusTone(c.status)">{{ c.status }}</AppBadge></td>
            <td class="px-4 py-2 text-right">
              <AppButton v-if="c.status === 'UNUSED'" variant="ghost" @click="onRevoke(c.id)">Revocar</AppButton>
            </td>
          </tr>
          <tr v-if="!loading && codes.length === 0">
            <td colspan="4" class="px-4 py-8 text-center text-gray-400">Sin códigos todavía.</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
