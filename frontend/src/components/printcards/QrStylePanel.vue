<script setup lang="ts">
/**
 * Styling for the business's shared QR — the same qr_codes row their public
 * profile, loyalty card and every printed card all use.
 *
 * It lives beside the designer rather than in the properties panel because it
 * is deliberately NOT a property of any one element: restyling it changes
 * every QR the business has, which is the point. Extracted from the old
 * editor form so the designer can host it without inheriting the rest of that
 * form's per-template fields.
 */
import { ref, watch } from 'vue'

import * as adminQrApi from '@/api/adminQr'
import type { QrCode } from '@/types'

const props = defineProps<{ clientId: string }>()
/** Emitted after a style change has been persisted, so the canvas can refetch its QR artwork. */
const emit = defineEmits<{ changed: [] }>()

const clientQr = ref<QrCode | null>(null)
const warnings = ref<string[]>([])
const loading = ref(false)
const logoPreviewUrl = ref('')
const logoUploading = ref(false)
const saveStatus = ref<'idle' | 'saving' | 'saved'>('idle')

let debounceTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

async function load() {
  if (clientQr.value) return
  loading.value = true
  try {
    clientQr.value = await adminQrApi.getClientQr(props.clientId)
    logoPreviewUrl.value = clientQr.value.logo_url ?? ''
    await refreshWarnings()
  } finally {
    loading.value = false
  }
}

async function refreshWarnings() {
  const validation = await adminQrApi.validateClientQr(props.clientId)
  warnings.value = validation.warnings
}

/**
 * Bumped on every debounce firing so a slow, superseded save can tell it is
 * stale and skip applying its older result — two overlapping cycles could
 * otherwise resolve out of order and flash an earlier color back onto the
 * canvas.
 */
let saveGeneration = 0

function scheduleSave() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(async () => {
    if (!clientQr.value) return
    const generation = ++saveGeneration
    saveStatus.value = 'saving'
    await adminQrApi.updateClientQr(props.clientId, {
      foreground_color: clientQr.value.foreground_color,
      background_color: clientQr.value.background_color,
      module_style: clientQr.value.module_style,
      eye_style: clientQr.value.eye_style,
      logo_media_id: clientQr.value.logo_media_id,
      logo_style: clientQr.value.logo_style,
      eye_color_from_logo: clientQr.value.eye_color_from_logo,
      preset_icon: clientQr.value.preset_icon,
      frame_shape: clientQr.value.frame_shape,
      shape_fill: clientQr.value.shape_fill,
    })
    if (generation !== saveGeneration) return
    saveStatus.value = 'saved'
    if (hideTimer) clearTimeout(hideTimer)
    hideTimer = setTimeout(() => (saveStatus.value = 'idle'), 1500)
    await refreshWarnings()
    emit('changed')
  }, 400)
}

watch(clientQr, () => scheduleSave(), { deep: true })

async function onLogoFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !clientQr.value) return
  logoUploading.value = true
  try {
    const media = await adminQrApi.uploadClientMedia(props.clientId, file)
    logoPreviewUrl.value = media.file_path
    // Uploading a logo makes the QR immersive by default — that is the whole
    // point of this control; toggleImmersive is the opt-out.
    clientQr.value.logo_media_id = media.id
    clientQr.value.preset_icon = null
    clientQr.value.frame_shape = 'custom_logo'
    clientQr.value.shape_fill = true
  } finally {
    logoUploading.value = false
  }
}

function toggleImmersive() {
  if (!clientQr.value) return
  if (clientQr.value.frame_shape === 'custom_logo') {
    clientQr.value.frame_shape = null
    clientQr.value.shape_fill = false
  } else {
    clientQr.value.frame_shape = 'custom_logo'
    clientQr.value.shape_fill = true
  }
}

function clearBadge() {
  if (!clientQr.value) return
  logoPreviewUrl.value = ''
  clientQr.value.logo_media_id = null
  clientQr.value.preset_icon = null
  if (clientQr.value.frame_shape === 'custom_logo') {
    clientQr.value.frame_shape = null
    clientQr.value.shape_fill = false
  }
}

async function onDownload(format: 'png' | 'svg') {
  await adminQrApi.downloadClientQrExport(props.clientId, format)
}

defineExpose({ load })
</script>

<template>
  <details class="border-t border-gray-200" @toggle="load">
    <summary class="flex cursor-pointer select-none items-center justify-between px-3 py-2.5 text-sm font-medium text-gray-800">
      <span>Estilo del QR</span>
      <span class="text-xs font-normal">
        <span v-if="saveStatus === 'saving'" class="text-gray-400">Guardando…</span>
        <span v-else-if="saveStatus === 'saved'" class="text-green-600">Guardado</span>
      </span>
    </summary>

    <div class="space-y-4 border-t border-gray-100 px-3 pb-4 pt-2">
      <p class="text-xs text-gray-500">
        Es el QR del negocio: el mismo que usan su perfil público, su tarjeta de lealtad y todas sus tarjetas
        impresas. Los cambios se ven al instante en el lienzo.
      </p>

      <div v-if="loading && !clientQr" class="py-2 text-xs text-gray-400">Cargando…</div>

      <template v-if="clientQr">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600">Color del código</label>
            <input v-model.lazy="clientQr.foreground_color" type="color" class="h-9 w-full rounded-lg border border-gray-300" />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600">Color de fondo</label>
            <input v-model.lazy="clientQr.background_color" type="color" class="h-9 w-full rounded-lg border border-gray-300" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600">Módulos</label>
            <select v-model="clientQr.module_style" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
              <option value="square">Cuadrado</option>
              <option value="dots">Puntos</option>
              <option value="rounded">Redondeado</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600">Ojos</label>
            <select v-model="clientQr.eye_style" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
              <option value="square">Cuadrado</option>
              <option value="circular">Circular</option>
              <option value="rounded">Redondeado</option>
            </select>
          </div>
        </div>

        <div class="space-y-2">
          <label class="mb-1.5 block text-xs font-medium text-gray-600">Logo del negocio</label>
          <div class="flex items-center gap-3">
            <label class="relative shrink-0 cursor-pointer">
              <img v-if="logoPreviewUrl" :src="logoPreviewUrl" alt="Logo" class="h-12 w-12 rounded-lg border border-gray-200 object-cover" />
              <div
                v-else
                class="flex h-12 w-12 items-center justify-center rounded-lg border border-dashed border-gray-300 px-1 text-center text-[10px] text-gray-400"
              >
                {{ logoUploading ? '…' : 'Subir' }}
              </div>
              <input
                type="file"
                accept="image/png,image/jpeg,image/gif"
                class="hidden"
                :disabled="logoUploading"
                @change="onLogoFileSelected"
              />
            </label>
            <p class="flex-1 text-xs text-gray-400">
              El QR toma la forma de tu logo — inmerso en el propio código, no un ícono pegado encima.
            </p>
          </div>

          <label v-if="clientQr.logo_media_id" class="flex items-start gap-2 pt-1">
            <input type="checkbox" :checked="clientQr.frame_shape === 'custom_logo'" class="mt-0.5" @change="toggleImmersive" />
            <span class="text-xs text-gray-600">Hacer el QR inmersivo con este logo</span>
          </label>

          <button v-if="clientQr.logo_media_id" type="button" class="text-xs text-gray-500 hover:underline" @click="clearBadge">
            Quitar logo
          </button>
        </div>

        <div v-if="warnings.length" class="space-y-1 rounded-lg border border-amber-200 bg-amber-50 p-2.5">
          <p v-for="(w, i) in warnings" :key="i" class="text-xs text-amber-800">⚠ {{ w }}</p>
        </div>

        <div class="flex gap-3">
          <button type="button" class="text-xs text-gray-500 hover:underline" @click="onDownload('png')">Descargar QR (PNG)</button>
          <button type="button" class="text-xs text-gray-500 hover:underline" @click="onDownload('svg')">Descargar QR (SVG)</button>
        </div>
      </template>
    </div>
  </details>
</template>
