<script setup lang="ts">
/**
 * The properties panel. It renders whatever elementSchemas declares for the
 * selected element's TYPE — it has no knowledge of card designs, headlines or
 * discount codes, which is exactly what the old per-template form had wired
 * into it.
 *
 * Two things are not schema-driven, deliberately: geometry (shared by every
 * type, so it is always shown) and a QR's destination (which needs the
 * client's own list of reachable targets, so it reuses QrTargetSelect).
 */
import { computed, ref } from 'vue'

import { useCardEditorStore } from '@/stores/cardEditor'
import type { ElementProps } from '@/types/cardLayout'
import type { QrTargetOption, QrTargetType } from '@/api/printCards'
import * as adminQrApi from '@/api/adminQr'
import ColorInput from '@/components/editor/ColorInput.vue'
import QrTargetSelect from '@/components/editor/QrTargetSelect.vue'
import { groupedSchema, type PropertyField } from './elementSchemas'
import { resolveCaptionState } from './captionSegments'

const props = defineProps<{ qrTargets: QrTargetOption[]; clientId: string; logoUrl?: string }>()

const store = useCardEditorStore()

const el = computed(() => store.singleSelected)
const groups = computed(() => (el.value ? groupedSchema(el.value.type, el.value.props) : []))

function value(field: PropertyField): unknown {
  return el.value?.props[field.key]
}

function update(field: PropertyField, raw: unknown) {
  if (!el.value) return
  let next = raw
  if (field.kind === 'number' || field.kind === 'range') {
    const parsed = Number(raw)
    if (Number.isNaN(parsed)) return
    next = parsed
  }
  // The options list stores numbers for things like font weight; a <select>
  // hands back strings, so coerce using the option's own declared type.
  if (field.kind === 'select' && field.options) {
    const match = field.options.find((o) => String(o.value) === String(raw))
    if (match) next = match.value
  }
  store.setProps(el.value.id, { [field.key]: next } as ElementProps)
}

function updateGeometry(key: 'x' | 'y' | 'w' | 'h' | 'rotation', raw: string) {
  if (!el.value) return
  const parsed = Number(raw)
  if (Number.isNaN(parsed)) return
  store.setElement(el.value.id, { [key]: parsed })
}

function updateOpacity(raw: string) {
  if (!el.value) return
  const parsed = Number(raw)
  if (Number.isNaN(parsed)) return
  store.setElement(el.value.id, { opacity: parsed })
}

function updateName(raw: string) {
  if (!el.value) return
  store.setElement(el.value.id, { name: raw })
}

function updateQrTargetType(targetType: QrTargetType) {
  if (!el.value) return
  // Switching to a target that carries no value (the profile, the loyalty
  // card) must clear the old block id, or the QR would keep pointing at
  // whatever was selected before.
  const keepsValue = targetType === 'block' || targetType === 'custom_url'
  store.setProps(el.value.id, {
    target_type: targetType,
    target_value: keepsValue ? (el.value.props.target_value ?? '') : null,
  })
}

function updateQrTargetValue(targetValue: string) {
  if (!el.value) return
  store.setProps(el.value.id, { target_value: targetValue || null })
}

// --- legacy embedded QR caption -------------------------------------------
//
// Toca/Escanea used to live baked into a QR element's own props; a card
// designed before that split still has one showing there. There is no UI
// left to CREATE one — only to remove it, so the business can replace it
// with independent prompt elements from the toolbar instead, the same way
// every QR from today onward already works.
const legacyCaptionShowing = computed(() => {
  if (!el.value || el.value.type !== 'qr') return false
  const state = resolveCaptionState(el.value.props)
  return state.showTap || state.showScan
})

function clearLegacyCaption() {
  if (!el.value) return
  store.setProps(el.value.id, { caption_mode: 'none' })
}

// --- image upload -----------------------------------------------------------
//
// An Image element's own source picker used to just be a "logo or media"
// dropdown with nothing behind "media" — there was no way to actually get a
// picture in. This uploads straight to the client's media library and wires
// the result onto the element, the same id+url pairing ProfileBlock already
// carries, so the canvas can render it with no extra fetch.
const imageUploading = ref(false)

async function onImageFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !el.value) return
  imageUploading.value = true
  try {
    const media = await adminQrApi.uploadClientMedia(props.clientId, file)
    store.setProps(el.value.id, { source: 'media', media_id: media.id, media_url: media.file_path })
  } finally {
    imageUploading.value = false
  }
}

function useBusinessLogo() {
  if (!el.value) return
  store.setProps(el.value.id, { source: 'logo' })
}

const imagePreviewUrl = computed(() => {
  if (!el.value || el.value.type !== 'image') return ''
  const p = el.value.props
  if (p.source === 'logo') return props.logoUrl ?? ''
  return p.media_url ?? ''
})

/** Whether a value is one of a field's curated presets, vs. something the business typed — drives the 'phrase' field kind's select/custom split. */
function isPreset(value: string, options: PropertyField['options']): boolean {
  return (options ?? []).some((o) => String(o.value) === value)
}

/** Rounded for display only — the stored value keeps its full precision. */
function display(n: number | undefined): string {
  if (n === undefined) return ''
  return String(Math.round(n * 100) / 100)
}
</script>

<template>
  <div class="flex h-full flex-col overflow-y-auto">
    <p v-if="store.selected.length === 0" class="px-4 py-8 text-center text-sm text-gray-400">
      Selecciona un elemento del lienzo para editarlo.<br />
      <span class="text-xs">Para el fondo, las esquinas o el logo, abre la pestaña «Tarjeta».</span>
    </p>

    <p v-else-if="!el" class="px-4 py-8 text-center text-sm text-gray-400">
      {{ store.selected.length }} elementos seleccionados. Puedes moverlos y redimensionarlos juntos; para editar
      propiedades, selecciona uno solo.
    </p>

    <div v-else class="space-y-5 p-4">
      <!-- Layer name -->
      <div>
        <label class="mb-1 block text-xs font-medium text-gray-600">Nombre de la capa</label>
        <input
          :value="el.name ?? ''"
          type="text"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          placeholder="Opcional"
          @input="updateName(($event.target as HTMLInputElement).value)"
        />
      </div>

      <!-- Geometry: identical for every type, which is what makes drag,
           resize and rotate uniform across the whole tree. -->
      <section>
        <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Posición y tamaño</h4>
        <div class="grid grid-cols-2 gap-2">
          <label v-for="key in (['x', 'y', 'w', 'h'] as const)" :key="key" class="block">
            <span class="mb-1 block text-xs font-medium text-gray-600">{{ key.toUpperCase() }}</span>
            <input
              :value="display(el[key])"
              type="number"
              step="0.5"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm tabular-nums"
              @change="updateGeometry(key, ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium text-gray-600">Rotación °</span>
            <input
              :value="display(el.rotation)"
              type="number"
              step="1"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm tabular-nums"
              @change="updateGeometry('rotation', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="block">
            <span class="mb-1 block text-xs font-medium text-gray-600">Opacidad</span>
            <input
              :value="el.opacity ?? 1"
              type="range"
              min="0"
              max="1"
              step="0.05"
              class="w-full"
              @input="updateOpacity(($event.target as HTMLInputElement).value)"
            />
          </label>
        </div>
      </section>

      <!-- A QR's destination needs the client's real target list, so it is the
           one field the schema cannot describe on its own. -->
      <section v-if="el.type === 'qr'">
        <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Destino</h4>
        <QrTargetSelect
          :options="props.qrTargets"
          :target-type="(el.props.target_type as QrTargetType) ?? 'profile'"
          :target-value="el.props.target_value ?? ''"
          @update:target-type="updateQrTargetType"
          @update:target-value="updateQrTargetValue"
        />
        <p v-if="el.props.legacy_slot !== undefined" class="mt-1.5 text-xs text-amber-600">
          Este QR conserva el enlace corto de exportaciones anteriores, así que cualquier tarjeta ya impresa lo sigue
          reconociendo. Cambiar su destino es seguro; duplicarlo crea un QR nuevo, sin esos escaneos.
        </p>
      </section>

      <!-- A card designed before Toca/Escanea became independent elements
           can still have one embedded in its QR. Nothing here creates a new
           one — only clears it, so it can be replaced with real prompt
           elements from the toolbar. -->
      <section v-if="el.type === 'qr' && legacyCaptionShowing" class="rounded-lg border border-amber-200 bg-amber-50 p-3">
        <p class="text-xs text-amber-800">
          Este QR conserva una leyenda «Toca/Escanea» incluida en el propio código, de un diseño anterior. Ya no se
          edita aquí — quítala y añade «Toca» o «Escanea» desde la barra de herramientas, como elementos
          independientes que puedes mover a donde quieras.
        </p>
        <button type="button" class="mt-2 text-xs font-medium text-amber-900 underline" @click="clearLegacyCaption">
          Quitar leyenda antigua
        </button>
      </section>

      <!-- An image's picture is an action (upload, or fall back to the
           business logo), not a value the generic schema loop can offer. -->
      <section v-if="el.type === 'image'">
        <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Imagen</h4>
        <div class="flex items-center gap-3">
          <label class="relative shrink-0 cursor-pointer">
            <img
              v-if="imagePreviewUrl"
              :src="imagePreviewUrl"
              alt=""
              class="h-14 w-14 rounded-lg border border-gray-200 object-cover"
            />
            <div
              v-else
              class="flex h-14 w-14 items-center justify-center rounded-lg border-2 border-dashed border-gray-300 text-center text-[10px] text-gray-400"
            >
              {{ imageUploading ? '…' : 'Subir' }}
            </div>
            <input
              type="file"
              accept="image/*"
              class="hidden"
              :disabled="imageUploading"
              @change="onImageFileSelected"
            />
          </label>
          <div class="flex-1 space-y-1.5">
            <p class="text-xs text-gray-500">
              {{
                el.props.source === 'logo'
                  ? 'Usando el logo del negocio.'
                  : el.props.media_url
                    ? 'Imagen subida para este elemento.'
                    : 'Sin imagen todavía — sube una o usa el logo.'
              }}
            </p>
            <p v-if="el.props.source === 'logo' && !imagePreviewUrl" class="text-xs text-amber-600">
              El negocio no tiene logo. Súbelo desde la pestaña «Tarjeta».
            </p>
            <button
              v-if="el.props.source !== 'logo'"
              type="button"
              class="text-xs font-medium text-indigo-600 hover:underline"
              @click="useBusinessLogo"
            >
              Usar el logo del negocio
            </button>
          </div>
        </div>
      </section>

      <!-- Everything else comes from the type's declared schema. -->
      <section v-for="group in groups" :key="group.group">
        <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">{{ group.group }}</h4>
        <div class="space-y-3">
          <div v-for="field in group.fields" :key="String(field.key)">
            <label class="mb-1 block text-xs font-medium text-gray-600">{{ field.label }}</label>

            <textarea
              v-if="field.kind === 'multiline'"
              :value="(value(field) as string) ?? ''"
              rows="3"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              @input="update(field, ($event.target as HTMLTextAreaElement).value)"
            />

            <ColorInput
              v-else-if="field.kind === 'color'"
              :model-value="(value(field) as string) ?? '#000000'"
              @update:model-value="update(field, $event)"
            />

            <select
              v-else-if="field.kind === 'select'"
              :value="String(value(field) ?? '')"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              @change="update(field, ($event.target as HTMLSelectElement).value)"
            >
              <option v-for="opt in field.options" :key="String(opt.value)" :value="String(opt.value)">
                {{ opt.label }}
              </option>
            </select>

            <!-- A curated dropdown that never blocks a custom value: pick one
                 of the presets, or "Personalizado…" to reveal a plain text
                 field seeded with whatever is already there. -->
            <template v-else-if="field.kind === 'phrase'">
              <select
                :value="isPreset((value(field) as string) ?? '', field.options) ? (value(field) as string) : '__custom__'"
                class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                @change="
                  (e) => {
                    const v = (e.target as HTMLSelectElement).value
                    if (v !== '__custom__') update(field, v)
                  }
                "
              >
                <option v-for="opt in field.options" :key="String(opt.value)" :value="String(opt.value)">
                  {{ opt.label }}
                </option>
                <option value="__custom__">Personalizado…</option>
              </select>
              <input
                v-if="!isPreset((value(field) as string) ?? '', field.options)"
                :value="(value(field) as string) ?? ''"
                type="text"
                class="mt-1.5 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                @input="update(field, ($event.target as HTMLInputElement).value)"
              />
            </template>

            <label v-else-if="field.kind === 'toggle'" class="flex items-center gap-2 text-sm text-gray-700">
              <input
                type="checkbox"
                class="h-4 w-4"
                :checked="Boolean(value(field))"
                @change="update(field, ($event.target as HTMLInputElement).checked)"
              />
              <span>Activado</span>
            </label>

            <input
              v-else-if="field.kind === 'range'"
              :value="Number(value(field) ?? 0)"
              type="range"
              :min="field.min"
              :max="field.max"
              :step="field.step"
              class="w-full"
              @input="update(field, ($event.target as HTMLInputElement).value)"
            />

            <input
              v-else-if="field.kind === 'number'"
              :value="display(value(field) as number)"
              type="number"
              :min="field.min"
              :max="field.max"
              :step="field.step"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm tabular-nums"
              @change="update(field, ($event.target as HTMLInputElement).value)"
            />

            <input
              v-else
              :value="(value(field) as string) ?? ''"
              type="text"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              @input="update(field, ($event.target as HTMLInputElement).value)"
            />

            <p v-if="field.hint" class="mt-0.5 text-xs text-gray-400">{{ field.hint }}</p>
          </div>
        </div>
      </section>

      <!-- Order controls sit here as well as in the layers panel: adjusting
           depth is something you want right where you are editing. -->
      <section>
        <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-400">Orden</h4>
        <div class="flex gap-2">
          <button
            type="button"
            class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-xs font-medium hover:bg-gray-50"
            @click="store.bringToFront(el.id)"
          >
            Traer al frente
          </button>
          <button
            type="button"
            class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-xs font-medium hover:bg-gray-50"
            @click="store.sendToBack(el.id)"
          >
            Enviar al fondo
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
