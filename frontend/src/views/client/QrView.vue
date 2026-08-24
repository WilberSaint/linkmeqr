<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import type { QrCode } from '@/types'
import * as qrApi from '@/api/qr'
import AppButton from '@/components/common/AppButton.vue'

const qr = ref<QrCode | null>(null)
const previewUrl = ref('')
const warnings = ref<string[]>([])
const loading = ref(true)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

async function load() {
  loading.value = true
  try {
    qr.value = await qrApi.getMyQr()
    await refreshPreview()
  } finally {
    loading.value = false
  }
}

async function refreshPreview() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = await qrApi.fetchQrPreview()
  const validation = await qrApi.validateMyQr()
  warnings.value = validation.warnings
}

function scheduleSave() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(async () => {
    if (!qr.value) return
    await qrApi.updateMyQr({
      foreground_color: qr.value.foreground_color,
      background_color: qr.value.background_color,
      module_style: qr.value.module_style,
      eye_style: qr.value.eye_style,
      logo_media_id: qr.value.logo_media_id,
    })
    await refreshPreview()
  }, 400)
}

watch(qr, scheduleSave, { deep: true })

async function onDownload(format: 'png' | 'svg') {
  await qrApi.downloadQrExport(format)
}

onMounted(load)
onUnmounted(() => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})
</script>

<template>
  <div class="p-6 max-w-3xl">
    <h1 class="text-lg font-semibold text-gray-900 mb-6">Mi código QR</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      <div v-if="qr" class="space-y-4">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Color del código</label>
            <input type="color" v-model="qr.foreground_color" class="w-full h-9 rounded-lg border border-gray-300" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Color de fondo</label>
            <input type="color" v-model="qr.background_color" class="w-full h-9 rounded-lg border border-gray-300" />
          </div>
        </div>

        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Estilo de módulos</label>
          <select v-model="qr.module_style" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
            <option value="square">Cuadrado</option>
            <option value="dots">Puntos</option>
            <option value="rounded">Redondeado</option>
          </select>
        </div>

        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Estilo de ojos</label>
          <select v-model="qr.eye_style" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
            <option value="square">Cuadrado</option>
            <option value="circular">Circular</option>
            <option value="rounded">Redondeado</option>
          </select>
        </div>

        <div v-if="warnings.length" class="bg-amber-50 border border-amber-200 rounded-lg p-3 space-y-1">
          <p v-for="(w, i) in warnings" :key="i" class="text-xs text-amber-800">⚠ {{ w }}</p>
        </div>

        <div class="flex gap-2 pt-2">
          <AppButton variant="secondary" @click="onDownload('png')">Descargar PNG</AppButton>
          <AppButton variant="secondary" @click="onDownload('svg')">Descargar SVG</AppButton>
        </div>
      </div>

      <div class="flex items-start justify-center">
        <div class="border border-gray-200 rounded-xl p-6 bg-white">
          <img v-if="previewUrl" :src="previewUrl" alt="QR preview" class="w-56 h-56" />
          <p v-else-if="loading" class="text-sm text-gray-400">Generando…</p>
        </div>
      </div>
    </div>
  </div>
</template>
