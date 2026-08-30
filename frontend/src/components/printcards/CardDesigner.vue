<script setup lang="ts">
/**
 * The print-card designer: a free-form canvas over the card's element tree.
 *
 * It replaces the old per-template form entirely. Nothing here knows what a
 * "Google review card" is — the six built-in designs only decide the tree a
 * card STARTS from, and from that moment the designer moves, restyles and
 * reorders whatever it likes.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowLeft, Image as ImageIcon, QrCode, Redo2, Save, ScanLine, Shapes, Smartphone, Star, Type, Undo2 } from '@lucide/vue'

import * as printCardsApi from '@/api/printCards'
import type { PrintCard, QrTargetOption } from '@/api/printCards'
import type { CardLayoutRevision } from '@/types/cardLayout'
import { useCardEditorStore } from '@/stores/cardEditor'
import type { ElementType } from '@/types/cardLayout'
import AppButton from '@/components/common/AppButton.vue'
import CanvasPanel from './CanvasPanel.vue'
import CardStage from './CardStage.vue'
import LayersPanel from './LayersPanel.vue'
import PropertiesPanel from './PropertiesPanel.vue'
import QrStylePanel from './QrStylePanel.vue'

const props = defineProps<{
  clientId: string
  card: PrintCard
  qrTargets: QrTargetOption[]
  logoUrl?: string
}>()
const emit = defineEmits<{ back: []; saved: []; logoChanged: [url: string] }>()

const store = useCardEditorStore()

/**
 * The right sidebar switches between the selected element's properties and the
 * card's own (background, corners, logo). Card-level settings used to have
 * nowhere to live at all, which made the background effectively uneditable.
 */
const sidebarTab = ref<'element' | 'card'>('element')

/** Selecting something on the canvas is a clear signal the designer wants its properties. */
watch(
  () => store.selectedIds.length,
  (count) => {
    if (count > 0) sidebarTab.value = 'element'
  },
)

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const conflict = ref(false)
const versions = ref<CardLayoutRevision[]>([])
const versionsOpen = ref(false)
const stageRef = ref<InstanceType<typeof CardStage> | null>(null)

const size = computed(() => printCardsApi.cardSizeInches(props.card))
const filenameBase = computed(() => props.card.title || props.card.layout_key || 'tarjeta')

// QR and the two prompts are deliberately NOT in this generic list: a QR
// needs addQRWithPrompts (it brings its own Toca/Escanea pair along), and
// Toca/Escanea are their own dedicated buttons so adding either is one
// click, not "add a prompt, then configure it to say Escanea".
const ADD_BUTTONS: { type: ElementType; label: string; icon: unknown }[] = [
  { type: 'text', label: 'Texto', icon: Type },
  { type: 'shape', label: 'Forma', icon: Shapes },
  { type: 'icon', label: 'Ícono', icon: Star },
  { type: 'image', label: 'Imagen', icon: ImageIcon },
]

async function loadLayout() {
  loading.value = true
  error.value = ''
  try {
    const res = await printCardsApi.getCardLayout(props.clientId, props.card.id)
    store.load(res.layout, res.layout_version)
  } catch {
    error.value = 'No se pudo cargar el diseño de esta tarjeta.'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!store.layout) return
  saving.value = true
  error.value = ''
  conflict.value = false
  try {
    const res = await printCardsApi.saveCardLayout(props.clientId, props.card.id, store.layout, store.baseVersion)
    store.markSaved(res.layout_version)
    emit('saved')
  } catch (e) {
    // A 409 means another admin saved this card while it was open. Silently
    // overwriting their work would be the wrong default for a design that
    // gets physically printed, so the designer is told and given the choice.
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 409) {
      conflict.value = true
      error.value = 'Alguien más guardó cambios en esta tarjeta mientras la editabas.'
    } else {
      error.value = 'No se pudo guardar el diseño.'
    }
  } finally {
    saving.value = false
  }
}

/** Discards the local edits and reloads whatever is now on the server. */
async function reloadAfterConflict() {
  conflict.value = false
  await loadLayout()
}

/** Keeps the local edits and re-saves them on top of the newer revision. */
async function overwriteAfterConflict() {
  const res = await printCardsApi.getCardLayout(props.clientId, props.card.id)
  store.markSaved(res.layout_version)
  conflict.value = false
  await save()
}

async function openVersions() {
  versionsOpen.value = !versionsOpen.value
  if (!versionsOpen.value) return
  versions.value = await printCardsApi.listCardLayoutVersions(props.clientId, props.card.id)
}

async function restore(version: number) {
  if (!confirm(`¿Restaurar la versión ${version}? Se guardará como una versión nueva, así que no perderás la actual.`)) return
  const res = await printCardsApi.restoreCardLayoutVersion(props.clientId, props.card.id, version)
  store.load(res.layout, res.layout_version)
  versionsOpen.value = false
  emit('saved')
}

// --- export -----------------------------------------------------------------

/**
 * Every export goes through the backend's own render of the SAVED tree, not
 * the canvas on screen — the exported file is the one that gets printed, and
 * it must reflect what was persisted (including the tracked /q/ links, which
 * only the server can mint).
 */
async function exportedSvg(): Promise<string | null> {
  if (store.dirty) {
    await save()
    // A refused save (a conflicting edit) means the server still holds a
    // different design. Exporting it anyway would hand over a file that does
    // not match the screen — and this file gets printed.
    if (store.dirty) return null
  }
  return printCardsApi.getPrintCardSvg(props.clientId, props.card.id)
}

const exporting = ref('')

// --- server-render check -----------------------------------------------------

/**
 * The canvas draws the tree in the browser so dragging is instant, which means
 * it is a twin of the Go exporter rather than the exporter itself. This shows
 * the server's own render of the CURRENT (possibly unsaved) tree, so a
 * designer can confirm what will actually print before committing — and so any
 * drift between the two renderers is visible rather than discovered on paper.
 */
const serverPreview = ref('')
const serverPreviewOpen = ref(false)
const serverPreviewLoading = ref(false)

async function toggleServerPreview() {
  serverPreviewOpen.value = !serverPreviewOpen.value
  if (!serverPreviewOpen.value || !store.layout) return
  serverPreviewLoading.value = true
  try {
    serverPreview.value = await printCardsApi.previewLayout(props.clientId, store.layout)
  } catch {
    serverPreview.value = ''
    error.value = 'No se pudo generar la vista del servidor.'
  } finally {
    serverPreviewLoading.value = false
  }
}

async function onExport(format: 'svg' | 'png' | 'pdf') {
  exporting.value = format
  error.value = ''
  try {
    const svg = await exportedSvg()
    if (svg === null) return
    if (format === 'svg') {
      printCardsApi.downloadSvgText(svg, `${filenameBase.value}.svg`)
    } else if (format === 'png') {
      await printCardsApi.downloadSvgAsPng(svg, `${filenameBase.value}.png`, size.value.widthIn, size.value.heightIn)
    } else {
      await printCardsApi.downloadSvgAsPdf(svg, `${filenameBase.value}.pdf`, size.value.widthIn, size.value.heightIn)
    }
  } catch {
    error.value = 'No se pudo generar el archivo.'
  } finally {
    exporting.value = ''
  }
}

// --- lifecycle --------------------------------------------------------------

function warnOnUnload(e: BeforeUnloadEvent) {
  if (!store.dirty) return
  e.preventDefault()
  e.returnValue = ''
}

onMounted(() => {
  loadLayout()
  window.addEventListener('beforeunload', warnOnUnload)
})

onBeforeUnmount(() => window.removeEventListener('beforeunload', warnOnUnload))

watch(() => props.card.id, loadLayout)

function onBack() {
  if (store.dirty && !confirm('Tienes cambios sin guardar. ¿Salir de todos modos?')) return
  emit('back')
}
</script>

<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col overflow-hidden rounded-xl border border-gray-200 bg-white">
    <!-- Toolbar -->
    <header class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-3 py-2">
      <button class="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700" @click="onBack">
        <ArrowLeft :size="14" /> Volver
      </button>

      <span class="mx-1 h-5 w-px bg-gray-200" />

      <button
        v-for="btn in ADD_BUTTONS"
        :key="btn.type"
        type="button"
        class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-700 transition hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-700"
        :title="`Añadir ${btn.label.toLowerCase()}`"
        @click="store.addElement(btn.type)"
      >
        <component :is="btn.icon" :size="15" />
        {{ btn.label }}
      </button>

      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-700 transition hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-700"
        title="Añadir un código QR, con Toca y Escanea debajo"
        @click="store.addQRWithPrompts()"
      >
        <QrCode :size="15" />
        QR
      </button>

      <span class="mx-1 h-5 w-px bg-gray-200" />

      <!-- Toca and Escanea, independent of any QR: each is one element you
           can drop anywhere, resize, restyle or delete on its own. -->
      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-700 transition hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-700"
        title="Añadir «Toca»"
        @click="store.addElement('prompt', undefined, 'tap')"
      >
        <Smartphone :size="15" />
        Toca
      </button>
      <button
        type="button"
        class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-700 transition hover:border-indigo-300 hover:bg-indigo-50 hover:text-indigo-700"
        title="Añadir «Escanea»"
        @click="store.addElement('prompt', undefined, 'scan')"
      >
        <ScanLine :size="15" />
        Escanea
      </button>

      <span class="mx-1 h-5 w-px bg-gray-200" />

      <button
        type="button"
        class="rounded p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-30"
        :disabled="!store.canUndo"
        title="Deshacer (Ctrl+Z)"
        @click="store.undo()"
      >
        <Undo2 :size="15" />
      </button>
      <button
        type="button"
        class="rounded p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-30"
        :disabled="!store.canRedo"
        title="Rehacer (Ctrl+Shift+Z)"
        @click="store.redo()"
      >
        <Redo2 :size="15" />
      </button>

      <div class="ml-auto flex items-center gap-2">
        <span v-if="store.dirty" class="text-xs text-amber-600">Sin guardar</span>
        <span v-else-if="store.baseVersion" class="text-xs text-gray-400">v{{ store.baseVersion }}</span>

        <button
          type="button"
          class="rounded px-2 py-1 text-xs hover:bg-gray-100"
          :class="serverPreviewOpen ? 'text-indigo-600' : 'text-gray-500'"
          title="Cómo lo renderiza el servidor (lo que realmente se imprime)"
          @click="toggleServerPreview"
        >
          Vista real
        </button>

        <button type="button" class="rounded px-2 py-1 text-xs text-gray-500 hover:bg-gray-100" @click="openVersions">
          Historial
        </button>

        <div class="flex items-center gap-1">
          <button
            v-for="fmt in (['svg', 'png', 'pdf'] as const)"
            :key="fmt"
            type="button"
            class="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-semibold uppercase text-gray-600 hover:bg-gray-50 disabled:opacity-50"
            :disabled="exporting !== ''"
            @click="onExport(fmt)"
          >
            {{ exporting === fmt ? '…' : fmt }}
          </button>
        </div>

        <AppButton :disabled="saving || !store.dirty" @click="save">
          <Save :size="14" class="mr-1 inline" />
          {{ saving ? 'Guardando…' : 'Guardar' }}
        </AppButton>
      </div>
    </header>

    <div v-if="error" class="flex items-center gap-3 border-b border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
      <span class="flex-1">{{ error }}</span>
      <template v-if="conflict">
        <button type="button" class="underline" @click="reloadAfterConflict">Cargar la versión guardada</button>
        <button type="button" class="underline" @click="overwriteAfterConflict">Conservar mis cambios</button>
      </template>
    </div>

    <div v-if="versionsOpen" class="max-h-40 overflow-y-auto border-b border-gray-200 bg-gray-50 px-3 py-2">
      <p v-if="versions.length === 0" class="text-xs text-gray-400">Sin versiones guardadas todavía.</p>
      <ul v-else class="space-y-1">
        <li v-for="v in versions" :key="v.id" class="flex items-center gap-3 text-xs">
          <span class="w-10 tabular-nums text-gray-500">v{{ v.version }}</span>
          <span class="flex-1 text-gray-400">{{ new Date(v.created_at).toLocaleString() }}</span>
          <button
            type="button"
            class="text-indigo-600 hover:underline disabled:text-gray-300"
            :disabled="v.version === store.baseVersion"
            @click="restore(v.version)"
          >
            Restaurar
          </button>
        </li>
      </ul>
    </div>

    <!-- Workspace -->
    <div v-if="loading" class="flex flex-1 items-center justify-center text-sm text-gray-400">Cargando diseño…</div>

    <div v-else class="flex min-h-0 flex-1">
      <aside class="w-64 shrink-0 overflow-y-auto border-r border-gray-200">
        <LayersPanel />
      </aside>

      <main class="min-w-0 flex-1">
        <CardStage ref="stageRef" :client-id="props.clientId" :logo-url="props.logoUrl" />
      </main>

      <aside class="flex w-80 shrink-0 flex-col overflow-y-auto border-l border-gray-200">
        <div v-if="serverPreviewOpen" class="border-b border-gray-200 p-3">
          <h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">Vista del servidor</h3>
          <div v-if="serverPreviewLoading" class="py-6 text-center text-xs text-gray-400">Generando…</div>
          <div v-else-if="serverPreview" class="rounded border border-gray-200 bg-gray-50 p-2" v-html="serverPreview"></div>
          <p class="mt-1.5 text-xs text-gray-400">Así se imprimirá. Vuelve a abrirla tras cambiar algo.</p>
        </div>

        <div class="flex shrink-0 border-b border-gray-200">
          <button
            v-for="tab in ([
              { key: 'element', label: 'Elemento' },
              { key: 'card', label: 'Tarjeta' },
            ] as const)"
            :key="tab.key"
            type="button"
            class="flex-1 border-b-2 px-3 py-2.5 text-sm font-medium transition"
            :class="
              sidebarTab === tab.key
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-gray-500 hover:bg-gray-50 hover:text-gray-700'
            "
            @click="sidebarTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </div>

        <PropertiesPanel
          v-show="sidebarTab === 'element'"
          class="flex-1"
          :qr-targets="props.qrTargets"
          :client-id="props.clientId"
          :logo-url="props.logoUrl"
        />

        <div v-show="sidebarTab === 'card'" class="flex-1">
          <CanvasPanel
            :client-id="props.clientId"
            :card="props.card"
            :logo-url="props.logoUrl"
            @logo-changed="emit('logoChanged', $event)"
          />
        </div>

        <QrStylePanel :client-id="props.clientId" @changed="stageRef?.refreshQrArtwork()" />
      </aside>
    </div>
  </div>
</template>
