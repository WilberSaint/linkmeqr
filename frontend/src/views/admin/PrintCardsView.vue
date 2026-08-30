<script setup lang="ts">
/**
 * LinkMeQR Studio — the admin's print-card workspace for one client.
 *
 * This view owns the list and the "choose a design" picker. Editing a card is
 * entirely CardDesigner's job: since the refactor, a card is an element tree,
 * so there is no per-design form here to keep in sync with the renderer.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Trash2, Download, Pencil, Plus, ArrowLeft, ScanLine } from '@lucide/vue'

import * as printCardsApi from '@/api/printCards'
import * as clientsApi from '@/api/clients'
import type {
  PrintCard,
  PrintCardContent,
  PrintCardLayoutKey,
  PrintCardPayload,
  PrintCardSaleStatus,
  PrintCardSizePreset,
  PrintCardStyle,
  QrTargetOption,
  QrTargetType,
} from '@/api/printCards'
import { PRINT_CARD_LAYOUTS, PRINT_CARD_SALE_STATUSES, PRINT_CARD_SIZES, PRINT_CARD_STYLES } from '@/api/printCards'
import AppButton from '@/components/common/AppButton.vue'
import CardDesigner from '@/components/printcards/CardDesigner.vue'

const route = useRoute()
const clientId = route.params.id as string

const clientName = ref('')
const profileLogoUrl = ref('')
const cards = ref<PrintCard[]>([])
const cardSvgs = ref<Record<string, string>>({})
const qrTargets = ref<QrTargetOption[]>([])
const loading = ref(true)
const creating = ref(false)

const mode = ref<'list' | 'pick-layout' | 'edit'>('list')
const editingCard = ref<PrintCard | null>(null)

const SOCIAL_BLOCK_TYPES = ['instagram', 'facebook', 'tiktok', 'youtube', 'whatsapp']

// The picker shows real rendered thumbnails at whatever physical size is
// selected, so "elige un diseño" conveys the actual printed format before
// committing. Cached per size so flipping the toggle doesn't re-render.
const pickLayoutSize = ref<PrintCardSizePreset>('business_card')
// Style is a second, independent dimension of the starting design: the same
// six arrangements look genuinely different as a colour block, a split card,
// a dark card with corner accents, or a framed one. Exposing it here turns six
// starting points into a few dozen.
const pickLayoutStyle = ref<PrintCardStyle>('block')
const layoutPreviewsCache = ref<Record<string, Record<PrintCardLayoutKey, string>>>({})
const layoutPreviewsLoading = ref(false)

/** Thumbnails are cached per (size, style) — both change what gets rendered. */
const previewCacheKey = computed(() => `${pickLayoutSize.value}|${pickLayoutStyle.value}`)
const layoutPreviews = computed(
  () => layoutPreviewsCache.value[previewCacheKey.value] ?? ({} as Record<PrintCardLayoutKey, string>),
)

// "Personalizado" has no fixed dimensions to preview at, so it is only
// offered later, once a card exists — here the point is comparing real
// formats side by side.
const pickLayoutSizeOptions = computed(
  () =>
    Object.fromEntries(Object.entries(PRINT_CARD_SIZES).filter(([key]) => key !== 'custom')) as Record<
      PrintCardSizePreset,
      (typeof PRINT_CARD_SIZES)[PrintCardSizePreset]
    >,
)

async function loadLayoutPreviews() {
  const key = previewCacheKey.value
  if (layoutPreviewsCache.value[key] || layoutPreviewsLoading.value) return
  layoutPreviewsLoading.value = true
  try {
    const entries = await Promise.all(
      PRINT_CARD_LAYOUTS.map(async (l) => {
        const content: PrintCardContent =
          l.key === 'multi_qr'
            ? { left_label: 'Síguenos', right_label: 'Reséñanos', left_target_type: 'profile', right_target_type: 'profile' }
            : { platform: 'instagram' }
        try {
          const svg = await printCardsApi.previewPrintCard(clientId, {
            layout_key: l.key,
            size_preset: pickLayoutSize.value,
            qr_target_type: 'profile',
            color_overrides: { style: pickLayoutStyle.value },
            content,
          })
          return [l.key, svg] as const
        } catch {
          return [l.key, ''] as const
        }
      }),
    )
    layoutPreviewsCache.value = {
      ...layoutPreviewsCache.value,
      [key]: Object.fromEntries(entries) as Record<PrintCardLayoutKey, string>,
    }
  } finally {
    layoutPreviewsLoading.value = false
  }
}

watch([pickLayoutSize, pickLayoutStyle], () => {
  if (mode.value === 'pick-layout') loadLayoutPreviews()
})

async function load() {
  loading.value = true
  try {
    const [client, list, targets] = await Promise.all([
      clientsApi.getClient(clientId),
      printCardsApi.listPrintCards(clientId),
      printCardsApi.getQrTargets(clientId),
    ])
    clientName.value = client.full_name
    cards.value = list
    qrTargets.value = targets
    try {
      const profile = await clientsApi.getClientProfile(clientId)
      profileLogoUrl.value = profile.logo_url ?? ''
    } catch {
      profileLogoUrl.value = ''
    }
    await refreshThumbnails()
  } finally {
    loading.value = false
  }
}

async function refreshThumbnails() {
  const entries = await Promise.all(
    cards.value.map(async (c) => {
      try {
        return [c.id, await printCardsApi.getPrintCardSvg(clientId, c.id)] as const
      } catch {
        return [c.id, ''] as const
      }
    }),
  )
  cardSvgs.value = Object.fromEntries(entries)
}

function openCreate() {
  mode.value = 'pick-layout'
  loadLayoutPreviews()
}

/**
 * Picks the most relevant real destination for a freshly chosen design — a
 * Google-review card defaults to that block if one exists, a social card to
 * whichever social block exists. Never defaults to something this client does
 * not have, since the picker only offers what is actually there.
 */
function bestTarget(wantBlockTypes: string[]): { type: QrTargetType; value: string } {
  const match = qrTargets.value.find((o) => o.target_type === 'block' && wantBlockTypes.includes(o.block_type ?? ''))
  if (match) return { type: 'block', value: match.target_value ?? '' }
  return { type: 'profile', value: '' }
}

/** The starting payload for a design — only ever used to seed the card's first element tree. */
function seedPayload(key: PrintCardLayoutKey): PrintCardPayload {
  const content: PrintCardContent = {}
  let targetType: QrTargetType = 'profile'
  let targetValue = ''

  if (key === 'google_review' || key === 'thank_you') {
    const t = bestTarget(['google_review'])
    targetType = t.type
    targetValue = t.value
    // A review card has no sensible implicit fallback: without a review
    // block on file it needs an explicit link.
    if (t.type === 'profile' && key === 'google_review') targetType = 'custom_url'
  } else if (key === 'menu_scan') {
    const t = bestTarget(['menu'])
    targetType = t.type
    targetValue = t.value
  } else if (key === 'loyalty_card') {
    targetType = 'loyalty'
  } else if (key === 'social_follow') {
    const t = bestTarget(SOCIAL_BLOCK_TYPES)
    targetType = t.type
    targetValue = t.value
    const match = qrTargets.value.find((o) => o.target_value === t.value && o.target_type === 'block')
    content.platform = match?.block_type ?? 'instagram'
  }

  if (key === 'multi_qr') {
    const follow = bestTarget(SOCIAL_BLOCK_TYPES)
    content.left_label = 'Síguenos'
    content.right_label = 'Reséñanos'
    content.left_target_type = follow.type === 'block' ? 'block' : 'profile'
    content.left_target_value = follow.value || undefined
    content.right_target_type = 'loyalty'
    targetType = 'profile'
    targetValue = ''
  }

  return {
    layout_key: key,
    title: null,
    size_preset: pickLayoutSize.value,
    custom_width_cm: null,
    custom_height_cm: null,
    qr_target_type: targetType,
    qr_target_value: targetType === 'block' || targetType === 'custom_url' ? targetValue || null : null,
    color_overrides: { style: pickLayoutStyle.value },
    content,
  }
}

/**
 * Choosing a design creates the card straight away. The designer edits a
 * persisted element tree with a real revision history, so it needs a card to
 * attach to — and an unwanted one is one click to delete from the list.
 */
async function pickLayout(key: PrintCardLayoutKey) {
  creating.value = true
  try {
    const card = await printCardsApi.createPrintCard(clientId, seedPayload(key))
    cards.value = [card, ...cards.value]
    openDesigner(card)
  } finally {
    creating.value = false
  }
}

function openDesigner(card: PrintCard) {
  editingCard.value = card
  mode.value = 'edit'
}

function backToList() {
  mode.value = 'list'
  editingCard.value = null
}

/** The designer can upload the business logo mid-edit; keep the view's copy current. */
async function onLogoChanged(url: string) {
  profileLogoUrl.value = url
  layoutPreviewsCache.value = {}
  await refreshThumbnails()
}

async function onDesignerSaved() {
  // The saved design changes the card's thumbnail and, if it was the first
  // save, its revision count — cheap enough to just refresh the list.
  await refreshThumbnails()
}

async function onDelete(c: PrintCard) {
  if (!confirm(`¿Eliminar la tarjeta "${c.title || layoutLabel(c.layout_key)}"?`)) return
  await printCardsApi.deletePrintCard(clientId, c.id)
  await load()
}

async function onChangeStatus(c: PrintCard, status: PrintCardSaleStatus) {
  let note = c.sale_note
  // Only ask for a note the moment it becomes relevant — a draft doesn't need
  // one, but "se imprimió"/"se entregó" is exactly when you'd jot down what
  // was charged or who picked it up.
  if (status !== 'draft' && status !== c.status) {
    const entered = prompt('Nota (opcional) — ej. cuánto se cobró, quién la recogió:', c.sale_note ?? '')
    if (entered === null) return
    note = entered || null
  }
  const updated = await printCardsApi.updatePrintCardStatus(clientId, c.id, status, note)
  const idx = cards.value.findIndex((x) => x.id === c.id)
  if (idx !== -1) cards.value[idx] = updated
}

function statusTone(status: PrintCardSaleStatus) {
  if (status === 'delivered') return 'bg-green-100 text-green-700'
  if (status === 'printed') return 'bg-amber-100 text-amber-700'
  return 'bg-gray-100 text-gray-600'
}

function layoutLabel(key: PrintCardLayoutKey) {
  return PRINT_CARD_LAYOUTS.find((l) => l.key === key)?.label ?? key
}

async function onDownload(c: PrintCard, format: 'svg' | 'png' | 'pdf') {
  const svg = cardSvgs.value[c.id] || (await printCardsApi.getPrintCardSvg(clientId, c.id))
  const name = c.title || c.layout_key
  const { widthIn, heightIn } = printCardsApi.cardSizeInches(c)
  if (format === 'svg') printCardsApi.downloadSvgText(svg, `${name}.svg`)
  else if (format === 'png') await printCardsApi.downloadSvgAsPng(svg, `${name}.png`, widthIn, heightIn)
  else await printCardsApi.downloadSvgAsPdf(svg, `${name}.pdf`, widthIn, heightIn)
}

onMounted(load)
</script>

<template>
  <div class="p-4 sm:p-6" :class="mode === 'edit' ? 'max-w-none' : 'max-w-5xl'">
    <RouterLink
      v-if="mode !== 'edit'"
      :to="{ name: 'admin-client-detail', params: { id: clientId } }"
      class="mb-4 inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
    >
      <ArrowLeft :size="14" /> {{ clientName || 'Cliente' }}
    </RouterLink>

    <!-- List -->
    <div v-if="mode === 'list'">
      <div class="mb-6 flex items-center justify-between gap-3">
        <h1 class="text-lg font-semibold text-gray-900">LinkMeQR Studio — {{ clientName }}</h1>
        <AppButton @click="openCreate"><Plus :size="15" /> Nueva tarjeta</AppButton>
      </div>
      <p class="mb-4 text-sm text-gray-500">
        Diseña tarjetas para imprimir y entregarle a este cliente (reseñas de Google, seguirlo en redes, escanear su
        menú, su tarjeta de lealtad...) usando su QR y sus colores de marca.
      </p>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="c in cards" :key="c.id" class="overflow-hidden rounded-xl border border-gray-200 bg-white">
          <div class="flex aspect-[3/2] items-center justify-center bg-gray-50 p-3" v-html="cardSvgs[c.id]"></div>
          <div class="p-3">
            <p class="truncate text-sm font-medium text-gray-900">{{ c.title || layoutLabel(c.layout_key) }}</p>
            <div class="flex items-center justify-between gap-2">
              <p class="truncate text-xs text-gray-400">
                {{ layoutLabel(c.layout_key) }} ·
                {{
                  c.size_preset === 'custom'
                    ? `${c.custom_width_cm} × ${c.custom_height_cm} cm`
                    : PRINT_CARD_SIZES[c.size_preset].label
                }}
              </p>
              <span
                class="inline-flex shrink-0 items-center gap-1 text-[11px] font-medium text-gray-500"
                :title="c.scan_count === 1 ? '1 escaneo registrado' : `${c.scan_count} escaneos registrados`"
              >
                <ScanLine :size="12" /> {{ c.scan_count }}
              </span>
            </div>

            <div class="mt-2 flex gap-1">
              <button
                v-for="s in PRINT_CARD_SALE_STATUSES"
                :key="s.value"
                type="button"
                class="flex-1 rounded-md px-1.5 py-1 text-[11px] font-medium transition"
                :class="c.status === s.value ? statusTone(s.value) : 'bg-gray-50 text-gray-400 hover:bg-gray-100'"
                @click="onChangeStatus(c, s.value)"
              >
                {{ s.label }}
              </button>
            </div>
            <p v-if="c.sale_note" class="mt-1 line-clamp-1 text-[11px] text-gray-400" :title="c.sale_note">
              {{ c.sale_note }}
            </p>

            <div class="mt-2 flex items-center gap-3">
              <button class="inline-flex items-center gap-1 text-xs text-indigo-600 hover:underline" @click="openDesigner(c)">
                <Pencil :size="12" /> Editar
              </button>
              <button
                v-for="fmt in (['svg', 'png', 'pdf'] as const)"
                :key="fmt"
                class="inline-flex items-center gap-1 text-xs uppercase text-gray-500 hover:underline"
                @click="onDownload(c, fmt)"
              >
                <Download :size="12" /> {{ fmt }}
              </button>
              <button class="ml-auto inline-flex items-center gap-1 text-xs text-red-500 hover:underline" @click="onDelete(c)">
                <Trash2 :size="12" />
              </button>
            </div>
          </div>
        </div>
        <p v-if="!loading && cards.length === 0" class="col-span-full py-8 text-center text-sm text-gray-400">
          Sin tarjetas todavía.
        </p>
      </div>
    </div>

    <!-- Design picker -->
    <div v-else-if="mode === 'pick-layout'">
      <button class="mb-4 inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700" @click="backToList">
        <ArrowLeft :size="14" /> Volver
      </button>
      <h1 class="mb-1 text-lg font-semibold text-gray-900">Elige un diseño</h1>
      <p class="mb-3 text-sm text-gray-500">
        Es sólo el punto de partida: al abrirlo podrás mover, cambiar y añadir lo que quieras.
      </p>
      <div class="mb-5 flex flex-wrap gap-4">
        <label class="block w-full max-w-xs">
          <span class="mb-1 block text-xs font-medium text-gray-600">Formato</span>
          <select v-model="pickLayoutSize" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option v-for="(s, key) in pickLayoutSizeOptions" :key="key" :value="key">{{ s.label }}</option>
          </select>
        </label>

        <label class="block w-full max-w-xs">
          <span class="mb-1 block text-xs font-medium text-gray-600">Estilo</span>
          <select v-model="pickLayoutStyle" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
            <option v-for="st in PRINT_CARD_STYLES" :key="st.value" :value="st.value">{{ st.label }}</option>
          </select>
          <span class="mt-1 block text-xs text-gray-400">
            {{ PRINT_CARD_STYLES.find((st) => st.value === pickLayoutStyle)?.description }}
          </span>
        </label>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <button
          v-for="l in PRINT_CARD_LAYOUTS"
          :key="l.key"
          type="button"
          class="overflow-hidden rounded-xl border border-gray-200 bg-white text-left transition hover:border-indigo-400 hover:shadow-md disabled:opacity-50"
          :disabled="creating"
          @click="pickLayout(l.key)"
        >
          <div
            class="flex items-center justify-center border-b border-gray-100 bg-gray-50 p-4"
            :style="{
              aspectRatio: `${PRINT_CARD_SIZES[pickLayoutSize].widthIn} / ${PRINT_CARD_SIZES[pickLayoutSize].heightIn}`,
            }"
          >
            <div v-if="layoutPreviews[l.key]" class="h-full w-full" v-html="layoutPreviews[l.key]"></div>
            <div v-else class="h-full w-full animate-pulse rounded bg-gray-100"></div>
          </div>
          <div class="p-3.5">
            <p class="text-sm font-semibold text-gray-900">{{ l.label }}</p>
            <p class="mt-1 text-xs text-gray-500">{{ l.description }}</p>
          </div>
        </button>
      </div>
    </div>

    <!-- Designer -->
    <CardDesigner
      v-else-if="mode === 'edit' && editingCard"
      :client-id="clientId"
      :card="editingCard"
      :qr-targets="qrTargets"
      :logo-url="profileLogoUrl"
      @back="backToList"
      @saved="onDesignerSaved"
      @logo-changed="onLogoChanged"
    />

  </div>
</template>
