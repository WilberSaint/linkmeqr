<script setup lang="ts">
import { ExternalLink } from '@lucide/vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Check, Copy, Download, RefreshCw, Search } from '@lucide/vue'
import * as loyaltyApi from '@/api/loyalty'
import type { LoyaltyCustomer, LoyaltyProgram } from '@/api/loyalty'
import AppButton from '@/components/common/AppButton.vue'
import AppBadge from '@/components/common/AppBadge.vue'

const qrPreviewUrl = ref('')

async function loadQrPreview() {
  if (qrPreviewUrl.value) URL.revokeObjectURL(qrPreviewUrl.value)
  qrPreviewUrl.value = await loyaltyApi.fetchLoyaltyQrPreview()
}

onUnmounted(() => {
  if (qrPreviewUrl.value) URL.revokeObjectURL(qrPreviewUrl.value)
})

const program = ref<LoyaltyProgram | null>(null)
const customers = ref<LoyaltyCustomer[]>([])
const loading = ref(true)
const saving = ref(false)
const copied = ref(false)

const form = ref({
  stamps_required: 10,
  mid_reward_enabled: false,
  mid_reward_stamps: 5,
  mid_reward_description: '',
  reward_description: '',
  is_active: true,
})

const midRewardError = computed(() =>
  form.value.mid_reward_enabled && form.value.mid_reward_stamps >= form.value.stamps_required
    ? 'Debe ser menor a los sellos requeridos para el premio final.'
    : '',
)

const loyaltyUrl = computed(() => (program.value ? `${window.location.origin}/loyalty/${program.value.loyalty_token}` : ''))

async function load() {
  loading.value = true
  try {
    ;[program.value, customers.value] = await Promise.all([loyaltyApi.getMyProgram(), loyaltyApi.listMyLoyaltyCustomers()])
    if (program.value) {
      form.value = {
        stamps_required: program.value.stamps_required,
        mid_reward_enabled: program.value.mid_reward_stamps != null,
        mid_reward_stamps: program.value.mid_reward_stamps ?? Math.max(1, Math.floor(program.value.stamps_required / 2)),
        mid_reward_description: program.value.mid_reward_description ?? '',
        reward_description: program.value.reward_description ?? '',
        is_active: program.value.is_active,
      }
    }
  } finally {
    loading.value = false
  }
}

function programPayload() {
  return {
    stamps_required: form.value.stamps_required,
    mid_reward_stamps: form.value.mid_reward_enabled ? form.value.mid_reward_stamps : null,
    mid_reward_description: form.value.mid_reward_enabled ? form.value.mid_reward_description || null : null,
    reward_description: form.value.reward_description || null,
    is_active: form.value.is_active,
  }
}

async function onSave() {
  if (midRewardError.value) return
  saving.value = true
  try {
    program.value = await loyaltyApi.updateMyProgram(programPayload())
  } finally {
    saving.value = false
  }
}

async function onRegenerateToken() {
  if (midRewardError.value) return
  if (!confirm('Esto invalida el enlace/tag NFC actual — tendrás que reprogramar el tag con el nuevo enlace. ¿Continuar?')) return
  saving.value = true
  try {
    program.value = await loyaltyApi.updateMyProgram({ ...programPayload(), regenerate_token: true })
    await loadQrPreview()
  } finally {
    saving.value = false
  }
}

async function onDownloadQr(format: 'png' | 'svg') {
  await loyaltyApi.downloadLoyaltyQrExport(format)
}

async function copyUrl() {
  await navigator.clipboard.writeText(loyaltyUrl.value)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

async function onStamp(c: LoyaltyCustomer) {
  try {
    await loyaltyApi.stampCustomer(c.id)
  } catch {
    alert('No se pudo agregar el sello — puede que su tarjeta ya esté completa. Actualizando la lista…')
  }
  await load()
}

async function onRedeem(c: LoyaltyCustomer) {
  if (!confirm(`¿Canjear el premio de ${c.full_name}? Esto reinicia su contador de sellos a 0.`)) return
  await loyaltyApi.redeemCustomer(c.id)
  await load()
}

const customerSearch = ref('')

/** Stamping the right person gets slow once the list is more than a screenful. */
const filteredCustomers = computed(() => {
  const q = customerSearch.value.trim().toLowerCase()
  if (!q) return customers.value
  return customers.value.filter(
    (c) => c.full_name.toLowerCase().includes(q) || (c.phone ?? '').toLowerCase().includes(q),
  )
})

function stampProgress(c: LoyaltyCustomer): number {
  const required = program.value?.stamps_required ?? 0
  if (required <= 0) return 0
  return Math.min(100, Math.round((c.stamps_count / required) * 100))
}

function stampBadgeTone(c: LoyaltyCustomer): 'green' | 'indigo' | 'gray' {
  const required = program.value?.stamps_required ?? Infinity
  if (c.stamps_count >= required) return 'green'
  const mid = program.value?.mid_reward_stamps
  if (mid != null && c.stamps_count >= mid) return 'indigo'
  return 'gray'
}

onMounted(() => {
  load()
  loadQrPreview()
})
</script>

<template>
  <div class="p-4 sm:p-6 max-w-3xl space-y-6">
    <AppPageHeader
      title="Tarjeta de lealtad"
      description="Tus clientes juntan sellos escaneando tu QR de lealtad. Tú los sellas desde aquí y ellos ven su avance en su propia tarjeta."
    >
      <template #actions>
        <a
          v-if="loyaltyUrl"
          :href="loyaltyUrl"
          target="_blank"
          rel="noopener"
          class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
        >
          <ExternalLink :size="14" /> Ver tarjeta pública
        </a>
      </template>
    </AppPageHeader>

    <!-- The mechanism was invisible: nothing on this screen said how a person
         ends up in the list below, or why they might disappear from it. -->
    <section class="rounded-xl border border-indigo-100 bg-indigo-50/60 p-5">
      <h2 class="mb-3 text-sm font-semibold text-indigo-900">Cómo funciona</h2>
      <ol class="space-y-2 text-sm text-indigo-800/90">
        <li><strong>1.</strong> Tu cliente escanea tu QR de lealtad (o toca tu tag NFC).</li>
        <li><strong>2.</strong> Pone su nombre y teléfono una sola vez. Ahí queda registrado.</li>
        <li><strong>3.</strong> Su tarjeta queda guardada en <em>su</em> teléfono y aparece abajo en tu lista.</li>
        <li><strong>4.</strong> Cada visita, tú le das «Sellar» desde aquí. Él ve el sello en su tarjeta.</li>
        <li><strong>5.</strong> Al completar los sellos, le das «Canjear» y el contador vuelve a cero.</li>
      </ol>
      <p class="mt-3 border-t border-indigo-100 pt-3 text-sm text-indigo-800/80">
        Su tarjeta se identifica por el navegador de su teléfono. Si cambia de teléfono o borra sus datos de
        navegación, tendrá que registrarse otra vez y empezará de cero — por eso conviene pedirle el teléfono al
        registrarse, para reconocerlo en la lista.
      </p>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-4 sm:p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-3">Configuración del programa</h2>
      <form class="space-y-3" @submit.prevent="onSave">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-medium text-gray-700 mb-1">Sellos para completar la tarjeta</label>
            <input v-model.number="form.stamps_required" type="number" min="1" max="100" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
          </div>
          <label class="flex items-center gap-2 text-sm text-gray-700 sm:self-end sm:pb-2.5">
            <input type="checkbox" v-model="form.is_active" />
            Programa activo
          </label>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Premio final (a los {{ form.stamps_required }} sellos)</label>
          <input v-model="form.reward_description" placeholder="Ej. Un café gratis" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>

        <div class="border-t border-gray-100 pt-3">
          <label class="flex items-center gap-2 text-sm text-gray-700 mb-2">
            <input type="checkbox" v-model="form.mid_reward_enabled" />
            Premio intermedio (opcional)
          </label>
          <div v-if="form.mid_reward_enabled" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-700 mb-1">A los cuántos sellos</label>
              <input v-model.number="form.mid_reward_stamps" type="number" min="1" :max="form.stamps_required - 1" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 mb-1">Premio intermedio</label>
              <input v-model="form.mid_reward_description" placeholder="Ej. Una galleta gratis" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
            </div>
          </div>
          <p v-if="midRewardError" class="text-xs text-red-600 mt-1.5">{{ midRewardError }}</p>
        </div>

        <AppButton type="submit" :disabled="saving || !!midRewardError">Guardar</AppButton>
      </form>
    </section>

    <section class="bg-white border border-gray-200 rounded-xl p-4 sm:p-5">
      <h2 class="text-sm font-semibold text-gray-900 mb-2">Código para sellar</h2>
      <p class="text-xs text-gray-500 mb-3">
        Imprime este código y pégalo en tu mostrador o mesas. Cuando un cliente lo escanea con la cámara de su
        celular, se abre su tarjeta de sellos y suma uno automáticamente — sin apps ni instalar nada.
      </p>
      <div class="flex flex-col sm:flex-row items-start gap-4">
        <img v-if="qrPreviewUrl" :src="qrPreviewUrl" alt="Código QR para sellar" class="w-32 h-32 rounded-lg border border-gray-200" />
        <div v-else class="w-32 h-32 rounded-lg border border-gray-200 bg-gray-50 animate-pulse shrink-0"></div>
        <div class="flex flex-row sm:flex-col gap-4 sm:gap-2 pt-1">
          <button type="button" class="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-600 hover:underline" @click="onDownloadQr('png')">
            <Download :size="13" /> Descargar PNG
          </button>
          <button type="button" class="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-600 hover:underline" @click="onDownloadQr('svg')">
            <Download :size="13" /> Descargar SVG
          </button>
        </div>
      </div>

      <div class="mt-4 pt-3 border-t border-gray-100">
        <p class="text-xs text-gray-400 mb-1.5">¿Ya tienes un tag NFC? Prográmalo con este mismo enlace:</p>
        <div class="flex items-center gap-2">
          <code class="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-lg px-3 py-2 truncate">{{ loyaltyUrl }}</code>
          <button type="button" class="p-2 rounded-lg border border-gray-300 text-gray-500 hover:text-gray-700 shrink-0" @click="copyUrl">
            <Check v-if="copied" :size="15" class="text-green-600" />
            <Copy v-else :size="15" />
          </button>
        </div>
        <button type="button" class="mt-2 inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-red-600" @click="onRegenerateToken">
          <RefreshCw :size="12" /> Regenerar enlace (invalida el código y el tag actuales)
        </button>
      </div>
    </section>

    <section class="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div class="border-b border-gray-100 px-4 py-3">
        <!-- Deliberately NOT just "Clientes": in this app that word already
             means the businesses in the admin panel. These are the people who
             visit this business and collect its stamps. -->
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-sm font-semibold text-gray-900">Quién está juntando sellos</h2>
            <p class="mt-0.5 text-xs text-gray-500">
              Las personas que visitan tu negocio y se registraron en tu tarjeta.
            </p>
          </div>
          <span class="text-xs text-gray-400">{{ customers.length }} registradas</span>
        </div>

        <div v-if="customers.length > 4" class="relative mt-3">
          <Search :size="15" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            v-model="customerSearch"
            type="text"
            placeholder="Buscar por nombre o teléfono…"
            class="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm"
          />
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th class="text-left px-4 py-2.5 font-medium">Nombre</th>
              <th class="text-left px-4 py-2.5 font-medium">Sellos</th>
              <th class="text-right px-4 py-2.5 font-medium">Acciones</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-for="c in filteredCustomers" :key="c.id">
              <td class="px-4 py-2.5 whitespace-nowrap">
                <p class="text-gray-900">{{ c.full_name }}</p>
                <p v-if="c.phone" class="text-xs text-gray-400">{{ c.phone }}</p>
              </td>
              <td class="whitespace-nowrap px-4 py-2.5">
                <div class="flex items-center gap-2">
                  <AppBadge :tone="stampBadgeTone(c)">
                    {{ c.stamps_count }} / {{ program?.stamps_required }}
                  </AppBadge>
                  <div class="h-1.5 w-20 overflow-hidden rounded-full bg-gray-100">
                    <div
                      class="h-full rounded-full transition-all"
                      :class="c.stamps_count >= (program?.stamps_required ?? Infinity) ? 'bg-amber-500' : 'bg-indigo-500'"
                      :style="{ width: `${stampProgress(c)}%` }"
                    />
                  </div>
                </div>
              </td>
              <td class="px-4 py-2.5 text-right whitespace-nowrap space-x-3">
                <button
                  v-if="c.stamps_count < (program?.stamps_required ?? Infinity)"
                  class="text-sm text-indigo-600 hover:underline"
                  @click="onStamp(c)"
                >
                  Sellar
                </button>
                <button
                  v-if="c.stamps_count >= (program?.stamps_required ?? Infinity)"
                  class="text-sm text-amber-600 hover:underline"
                  @click="onRedeem(c)"
                >
                  Canjear
                </button>
              </td>
            </tr>
            <tr v-if="!loading && customers.length > 0 && filteredCustomers.length === 0">
              <td colspan="3" class="px-4 py-8 text-center text-sm text-gray-400">Nadie coincide con tu búsqueda.</td>
            </tr>
            <tr v-if="!loading && customers.length === 0">
              <td colspan="3" class="px-4 py-10 text-center text-sm text-gray-400">
                Todavía nadie se ha registrado.<br />
                <span class="text-xs">
                  Comparte tu QR de lealtad o imprime una tarjeta con él para que empiecen a juntar sellos.
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
