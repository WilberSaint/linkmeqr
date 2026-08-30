<script setup lang="ts">
/**
 * The design canvas: the card drawn as SVG, with a transparent hit layer on
 * top that vue3-moveable and vue3-selecto manipulate.
 *
 * The split matters. Moveable and Selecto work on real DOM boxes, so each
 * element gets an invisible positioned <div> for them to grab; the visible
 * artwork stays a single SVG that mirrors the exporter exactly. Neither
 * library owns any state — they emit gestures, this component converts pixels
 * back into card units, and the store is the only thing that changes. That is
 * why every position on screen still comes from the element tree.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Moveable from 'vue3-moveable'
import Selecto from 'vue3-selecto'

import { useCardEditorStore } from '@/stores/cardEditor'
import type { CardElement } from '@/types/cardLayout'
import ElementRenderer from './ElementRenderer.vue'
import { useCardAssets } from './assetCache'
import { patternDef, type PatternKind } from './patterns'

const props = defineProps<{
  clientId: string
  /** Business logo URL, used by image elements whose source is "logo". */
  logoUrl?: string
}>()

const store = useCardEditorStore()
const { qrSvgs, iconSvgs, resolve, refetchQrs } = useCardAssets(props.clientId)

const viewportRef = ref<HTMLElement | null>(null)
const stageRef = ref<HTMLElement | null>(null)
const moveableRef = ref<InstanceType<typeof Moveable> | null>(null)
const selectoRef = ref<InstanceType<typeof Selecto> | null>(null)

const zoom = ref(1)
/** Pixels per card unit. The card is 100 units per inch, so 1 unit ≈ 1 CSS px at 100% on a 96dpi screen. */
const scale = computed(() => zoom.value)

const canvas = computed(() => store.layout?.canvas ?? { w: 350, h: 200, corner_r: 0, die_cut: null })

const stageStyle = computed(() => ({
  width: `${canvas.value.w * scale.value}px`,
  height: `${canvas.value.h * scale.value}px`,
}))

/** Elements bottom-first, matching paint order; hidden ones are skipped entirely. */
const visible = computed(() => store.elements.filter((el) => !el.hidden))

/** Only unlocked, visible elements get a hit box — locking is what makes an element unclickable. */
const hittable = computed(() => visible.value.filter((el) => !el.locked))

/**
 * Snap guides come from every OTHER element on the card. Including the
 * element currently being dragged or resized in its own guideline list is
 * what froze resizes solid: our hit-box position lags Moveable's internal
 * state by one Vue render tick, so the element's guideline briefly points at
 * where it JUST was, and every frame snaps the resize straight back there —
 * width and height stop moving until the gesture ends and the guideline
 * finally catches up.
 */
const guidelineTargets = computed(() => hittable.value.filter((el) => !store.selectedIds.includes(el.id)))

/**
 * Fixed guides for the card itself — its center and its edges — so an
 * element can snap to the middle of the card even when nothing else is
 * around to snap to. Positions are in pixels relative to `stageRef` (the
 * `container` given to Moveable below), the same frame the hit-boxes above
 * are placed in, so `w/2`/`h/2` here lines up exactly with a centered
 * element's own left/top.
 */
/**
 * vue3-moveable always mounts Moveable's own control box (and everything it
 * draws — handles, guideline lines) as a plain sibling of wherever the
 * <Moveable> tag sits, ignoring the `container`/`rootContainer` props for
 * that placement (it's `warpSelf: true` under the hood, a mode react/vue
 * moveable itself calls private). <Moveable> lives next to stageRef, both
 * children of viewportRef — so viewportRef, not stageRef, is where guideline
 * pixel positions actually get drawn from, no matter what `container` claims.
 * Rather than fight that, verticalGuidelines/horizontalGuidelines are kept in
 * viewportRef's own frame: the card's local position (0..w*scale) plus
 * stageRef's own offset within viewportRef (its offsetParent, since both
 * carry `position: relative`) — offsetLeft/offsetTop already fold in
 * viewportRef's padding and stageRef's mx-auto centering for free.
 */
const stageOffset = ref({ left: 0, top: 0 })
function syncStageOffset() {
  if (stageRef.value) stageOffset.value = { left: stageRef.value.offsetLeft, top: stageRef.value.offsetTop }
}

const verticalGuidelines = computed(() => {
  const w = canvas.value.w * scale.value
  const base = stageOffset.value.left
  return [base, base + w / 2, base + w]
})
const horizontalGuidelines = computed(() => {
  const h = canvas.value.h * scale.value
  const base = stageOffset.value.top
  return [base, base + h / 2, base + h]
})

const imageSrcs = computed<Record<string, string>>(() => {
  const out: Record<string, string> = {}
  for (const el of store.elements) {
    if (el.type !== 'image') continue
    if (el.props.source === 'logo' && props.logoUrl) out[el.id] = props.logoUrl
    else if (el.props.source === 'media' && el.props.media_url) out[el.id] = el.props.media_url
  }
  return out
})

const background = computed(() => store.layout?.background ?? { fill: '#ffffff' })

const backgroundFill = computed(() =>
  background.value.gradient_to ? 'url(#stageBg)' : background.value.fill || '#ffffff',
)

/**
 * The card's own texture, as a `<pattern>` def — mirrors backgroundPatternDef
 * in the Go renderer. This used to be entirely missing: the picker in the
 * Tarjeta tab wrote `background.pattern` to the store correctly, but nothing
 * ever read it back out when painting the SVG, so a chosen texture never
 * showed up here even though the real export already had it.
 */
const backgroundPattern = computed(() => {
  if (!canvas.value || !background.value.pattern) return ''
  const base = Math.min(canvas.value.w, canvas.value.h)
  const ink = background.value.pattern_ink || '#111827'
  return patternDef('cardPattern', background.value.pattern as PatternKind, base, ink)
})

// --- selection ↔ moveable targets ------------------------------------------

const targets = ref<HTMLElement[]>([])

/**
 * Moveable needs the actual DOM nodes for the current selection. They only
 * exist after the hit layer has rendered, so this waits a tick — otherwise
 * selecting a freshly added element would hand Moveable a stale or empty list
 * and the handles would not appear.
 *
 * Selecto keeps its OWN internal notion of what is selected, separate from
 * both Moveable's targets and our own store — it only ever updates that on
 * its own marquee/click gestures, never when the selection changes some
 * other way (adding an element, duplicating one, undo). Without telling it
 * explicitly here too, Selecto's next drag-start decision (see
 * onSelectoDragStart below) reasons from a selection that is one step
 * behind reality: right after adding an element, Selecto still thinks
 * nothing is selected, so the very first press on the new element — even
 * just to drag it into place — starts a fresh marquee instead of handing
 * off to Moveable, which can silently clear the selection Delete was about
 * to act on. A second, ordinary click then re-selects it correctly (that
 * click IS one of Selecto's own gestures, so its state catches up) — which
 * is the "has to be deselected and reselected before it can be deleted"
 * a freshly added element used to need.
 */
async function syncTargets() {
  await nextTick()
  const stage = stageRef.value
  if (!stage) {
    targets.value = []
    return
  }
  targets.value = store.selectedIds
    .map((id) => stage.querySelector<HTMLElement>(`[data-el-id="${id}"]`))
    .filter((n): n is HTMLElement => n !== null)
  moveableRef.value?.updateRect()
  selectoRef.value?.setSelectedTargets(targets.value)
}

watch(() => store.selectedIds.slice(), syncTargets, { deep: false })
watch(visible, syncTargets)
// Re-resolve artwork whenever an element that needs some appears or changes
// what it points at. The key includes the element id: two QRs with identical
// destinations are still two elements, each needing its own entry.
watch(
  () =>
    store.elements
      .map(
        (el) =>
          `${el.id}:${el.type}:${el.props.name}:${el.props.icon}:${el.props.color}:${el.props.target_type}:${el.props.target_value}:` +
          `${el.props.caption_mode}:${el.props.show_tap}:${el.props.tap_icon}:${el.props.show_scan}:${el.props.scan_icon}:` +
          `${el.props.caption}:${el.props.caption_fill}`,
      )
      .join('|'),
  () => resolve(store.elements),
)

// --- gesture → tree ---------------------------------------------------------

type Geometry = Pick<CardElement, 'x' | 'y' | 'w' | 'h' | 'rotation'>

/** Geometry captured when a gesture starts, so every frame applies a delta to the ORIGINAL box rather than compounding rounding errors. */
let gestureStart = new Map<string, Geometry>()

function beginGesture(ids: string[]) {
  // One history entry per gesture, not per frame: undo should step back over
  // the whole drag.
  store.snapshot()
  gestureStart = new Map()
  for (const id of ids) {
    const el = store.find(id)
    if (el) gestureStart.set(id, { x: el.x, y: el.y, w: el.w, h: el.h, rotation: el.rotation })
  }
}

/**
 * Returns the geometry a gesture started from, recording it on first sight if
 * the matching *Start event never arrived.
 *
 * That happens routinely: when a press both selects an element and begins
 * moving it, Moveable is started programmatically and does not re-emit
 * dragStart, so the frames that follow had no origin to apply their delta to
 * and the element simply refused to move. `applied` is how far the gesture has
 * already travelled by this first frame, which is what recovers the true
 * origin.
 */
function gestureOrigin(id: string, applied: { x: number; y: number }): Geometry | null {
  const known = gestureStart.get(id)
  if (known) return known

  const el = store.find(id)
  if (!el) return null
  store.snapshot()
  const origin: Geometry = {
    x: el.x - applied.x,
    y: el.y - applied.y,
    w: el.w,
    h: el.h,
    rotation: el.rotation,
  }
  gestureStart.set(id, origin)
  return origin
}

/** Gestures must not leak their origin into the next one. */
function endGesture() {
  gestureStart = new Map()
}

type MoveableTarget = HTMLElement | SVGElement

function idOf(target: MoveableTarget): string | null {
  return (target as HTMLElement).dataset?.elId ?? null
}

/**
 * Writes a gesture's geometry straight onto the target's own inline style,
 * synchronously, in the same tick Moveable computed it.
 *
 * Without this, resize was unusable: the hit box's CSS only ever changed one
 * Vue render tick after the store update, so every one of Moveable's OWN
 * internal per-frame calculations (which reads the target's live rect to
 * work out the next delta) ran against a box that hadn't caught up to the
 * previous frame yet. Two frames' worth of resize maths racing the same lag
 * produced the alternating "grows, snaps back, grows, snaps back" freeze —
 * dragging never showed it (translate deltas are self-contained, independent
 * of the target's current rect) but resizing depends on that rect every
 * frame. Vue's own reactive re-render from the store update below still
 * happens right after; it just repaints the same numbers, a no-op.
 */
function applyLive(target: MoveableTarget, geo: Partial<Geometry>) {
  const style = (target as HTMLElement).style
  if (geo.x !== undefined) style.left = `${geo.x * scale.value}px`
  if (geo.y !== undefined) style.top = `${geo.y * scale.value}px`
  if (geo.w !== undefined) style.width = `${geo.w * scale.value}px`
  if (geo.h !== undefined) style.height = `${geo.h * scale.value}px`
  if (geo.rotation !== undefined) style.transform = geo.rotation ? `rotate(${geo.rotation}deg)` : ''
}

function onDragStart(e: { target: MoveableTarget }) {
  const id = idOf(e.target)
  if (id) beginGesture([id])
}

function onDrag(e: { target: MoveableTarget; beforeTranslate: number[] }) {
  const id = idOf(e.target)
  if (!id) return
  const dx = e.beforeTranslate[0] / scale.value
  const dy = e.beforeTranslate[1] / scale.value
  const start = gestureOrigin(id, { x: dx, y: dy })
  if (!start) return
  applyLive(e.target, { x: start.x + dx, y: start.y + dy })
  store.setGeometry(id, { x: start.x + dx, y: start.y + dy })
}

function onDragGroupStart(e: { targets: MoveableTarget[] }) {
  beginGesture(e.targets.map(idOf).filter((id): id is string => id !== null))
}

function onDragGroup(e: { events: { target: MoveableTarget; beforeTranslate: number[] }[] }) {
  for (const ev of e.events) onDrag(ev)
}

function onResizeStart(e: { target: MoveableTarget }) {
  const id = idOf(e.target)
  if (id) beginGesture([id])
}

function onResize(e: { target: MoveableTarget; width: number; height: number; drag: { beforeTranslate: number[] } }) {
  const id = idOf(e.target)
  if (!id) return
  const dx = e.drag.beforeTranslate[0] / scale.value
  const dy = e.drag.beforeTranslate[1] / scale.value
  const start = gestureOrigin(id, { x: dx, y: dy })
  if (!start) return
  const geo = { x: start.x + dx, y: start.y + dy, w: e.width / scale.value, h: e.height / scale.value }
  applyLive(e.target, geo)
  store.setGeometry(id, geo)
}

function onResizeGroupStart(e: { targets: MoveableTarget[] }) {
  beginGesture(e.targets.map(idOf).filter((id): id is string => id !== null))
}

function onResizeGroup(e: { events: Parameters<typeof onResize>[0][] }) {
  for (const ev of e.events) onResize(ev)
}

function onRotateStart(e: { target: MoveableTarget }) {
  const id = idOf(e.target)
  if (id) beginGesture([id])
}

function onRotate(e: { target: MoveableTarget; rotation: number }) {
  const id = idOf(e.target)
  if (!id) return
  // Rotation is absolute, so it needs no origin — but it still has to open a
  // history entry the first time a gesture touches this element.
  gestureOrigin(id, { x: 0, y: 0 })
  applyLive(e.target, { rotation: e.rotation })
  store.setGeometry(id, { rotation: e.rotation })
}

function onRotateGroupStart(e: { targets: MoveableTarget[] }) {
  beginGesture(e.targets.map(idOf).filter((id): id is string => id !== null))
}

function onRotateGroup(e: { events: { target: MoveableTarget; rotation: number }[] }) {
  for (const ev of e.events) onRotate(ev)
}

// --- marquee selection ------------------------------------------------------

/**
 * Selecto and Moveable both want the pointer, so they have to hand it off
 * explicitly. Without this handler Selecto starts a marquee on EVERY press —
 * including presses on an element or on a resize handle — and Moveable never
 * receives the gesture, so the element does not follow the cursor and the
 * press appears to stick until the next click.
 */
function onSelectoDragStart(e: {
  inputEvent: MouseEvent
  stop: () => void
}) {
  const moveable = moveableRef.value
  const pressed = e.inputEvent.target as HTMLElement | null
  if (!pressed) return

  // A press on one of Moveable's own controls (resize corners, the rotation
  // handle) belongs to Moveable outright.
  if (moveable?.isMoveableElement?.(pressed)) {
    e.stop()
    return
  }
  // A press inside something already selected means "move this", not
  // "start a new marquee over it".
  if (targets.value.some((t) => t === pressed || t.contains(pressed))) {
    e.stop()
  }
}

function onSelectEnd(e: {
  selected: MoveableTarget[]
  isDragStart: boolean
  inputEvent: MouseEvent
}) {
  const ids = e.selected.map(idOf).filter((id): id is string => id !== null)
  store.selectMany(ids)

  // Pressing an unselected element selects it — and should begin moving it in
  // the same gesture, rather than making the user press once to select and
  // again to drag. Moveable needs its target list to have caught up first,
  // which is what waitToChangeTarget is for.
  if (!e.isDragStart) return
  e.inputEvent.preventDefault()
  nextTick(() => {
    const moveable = moveableRef.value
    if (!moveable) return
    const begin = () => moveable.dragStart(e.inputEvent)
    const waiting = moveable.waitToChangeTarget?.()
    if (waiting && typeof waiting.then === 'function') waiting.then(begin)
    else begin()
  })
}

// --- keyboard ---------------------------------------------------------------

function onKeydown(e: KeyboardEvent) {
  const target = e.target as HTMLElement | null
  // Never hijack keys while someone is typing into the properties panel.
  if (target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) return
  if (target?.isContentEditable) return

  const mod = e.ctrlKey || e.metaKey

  if (mod && e.key.toLowerCase() === 'z') {
    e.preventDefault()
    if (e.shiftKey) store.redo()
    else store.undo()
    return
  }
  if (mod && e.key.toLowerCase() === 'y') {
    e.preventDefault()
    store.redo()
    return
  }
  if (mod && e.key.toLowerCase() === 'd') {
    e.preventDefault()
    store.duplicate()
    return
  }
  if (e.key === 'Delete' || e.key === 'Backspace') {
    if (store.selectedIds.length === 0) return
    e.preventDefault()
    store.remove()
    return
  }
  // Arrow keys nudge by one card unit, or ten with shift — the fine
  // adjustment that dragging with a mouse cannot reach.
  const step = e.shiftKey ? 10 : 1
  const nudges: Record<string, [number, number]> = {
    ArrowLeft: [-step, 0],
    ArrowRight: [step, 0],
    ArrowUp: [0, -step],
    ArrowDown: [0, step],
  }
  const delta = nudges[e.key]
  if (delta && store.selectedIds.length > 0) {
    e.preventDefault()
    store.nudge(delta[0], delta[1])
    syncTargets()
  }
}

// --- zoom -------------------------------------------------------------------

function fitToViewport() {
  const vp = viewportRef.value
  if (!vp || !store.layout) return
  const padding = 48
  const availableW = vp.clientWidth - padding
  const availableH = vp.clientHeight - padding
  if (availableW <= 0 || availableH <= 0) return
  zoom.value = Math.min(availableW / canvas.value.w, availableH / canvas.value.h, 3)
}

function setZoom(next: number) {
  zoom.value = Math.min(4, Math.max(0.2, next))
}

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  resolve(store.elements)
  fitToViewport()
  nextTick(syncStageOffset)
  if (viewportRef.value && 'ResizeObserver' in window) {
    resizeObserver = new ResizeObserver(() => {
      moveableRef.value?.updateRect()
      syncStageOffset()
    })
    resizeObserver.observe(viewportRef.value)
    // stageRef's own box changes on every zoom (a CSS width/height driven by
    // `scale`, not by viewportRef resizing), which shifts its mx-auto offset
    // too — watch it directly so the fixed guidelines track it, not just
    // window/panel resizes.
    if (stageRef.value) resizeObserver.observe(stageRef.value)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  resizeObserver?.disconnect()
})

watch(() => store.layout, () => {
  resolve(store.elements)
  fitToViewport()
})

// zoom changes stageRef's own box (mx-auto recenters it) without
// necessarily resizing viewportRef, so it needs its own trigger too.
watch(scale, () => nextTick(syncStageOffset))

defineExpose({ fitToViewport, setZoom, zoom, refreshQrArtwork: () => refetchQrs(store.elements) })
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-2 border-b border-gray-200 bg-white px-3 py-2 text-xs">
      <button type="button" class="rounded px-2 py-1 hover:bg-gray-100" @click="setZoom(zoom - 0.1)">−</button>
      <span class="w-12 text-center tabular-nums text-gray-600">{{ Math.round(zoom * 100) }}%</span>
      <button type="button" class="rounded px-2 py-1 hover:bg-gray-100" @click="setZoom(zoom + 0.1)">+</button>
      <button type="button" class="rounded px-2 py-1 text-gray-600 hover:bg-gray-100" @click="fitToViewport">
        Ajustar
      </button>
      <span class="ml-auto text-gray-400">
        Supr borra · Ctrl+D duplica · Ctrl+Z deshace · flechas mueven
      </span>
    </div>

    <div ref="viewportRef" class="relative flex-1 overflow-auto bg-gray-100 p-6">
      <div ref="stageRef" class="relative mx-auto shadow-lg" :style="stageStyle">
        <!-- The visible card. Pointer events are off so every click reaches
             the hit layer above it. -->
        <svg
          class="pointer-events-none absolute inset-0 h-full w-full"
          :viewBox="`0 0 ${canvas.w} ${canvas.h}`"
          font-family="Arial, sans-serif"
        >
          <defs>
            <filter id="cardShadow" x="-60%" y="-60%" width="220%" height="220%">
              <feDropShadow dx="0" dy="3" stdDeviation="4" flood-color="#000000" flood-opacity="0.22" />
            </filter>
            <linearGradient v-if="background.gradient_to" id="stageBg" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" :stop-color="background.fill" />
              <stop offset="100%" :stop-color="background.gradient_to" />
            </linearGradient>
            <!-- eslint-disable-next-line vue/no-v-html -- generated pattern markup, not user input -->
            <g v-if="backgroundPattern" v-html="backgroundPattern" />
            <clipPath id="stageClip">
              <rect :width="canvas.w" :height="canvas.h" :rx="canvas.corner_r" />
            </clipPath>
            <mask v-if="canvas.die_cut" id="stageDieCut">
              <rect :width="canvas.w" :height="canvas.h" fill="#ffffff" />
              <circle :cx="canvas.die_cut.cx" :cy="canvas.die_cut.cy" :r="canvas.die_cut.r" fill="#000000" />
            </mask>
          </defs>

          <rect
            :width="canvas.w"
            :height="canvas.h"
            :rx="canvas.corner_r"
            :fill="backgroundFill"
            :mask="canvas.die_cut ? 'url(#stageDieCut)' : undefined"
          />
          <rect
            v-if="background.pattern"
            :width="canvas.w"
            :height="canvas.h"
            :rx="canvas.corner_r"
            fill="url(#cardPattern)"
            :mask="canvas.die_cut ? 'url(#stageDieCut)' : undefined"
          />

          <g clip-path="url(#stageClip)">
            <ElementRenderer
              v-for="el in visible"
              :key="el.id"
              :el="el"
              :qr-svgs="qrSvgs"
              :icon-svgs="iconSvgs"
              :image-srcs="imageSrcs"
            />
          </g>

          <circle
            v-if="canvas.die_cut"
            :cx="canvas.die_cut.cx"
            :cy="canvas.die_cut.cy"
            :r="canvas.die_cut.r"
            fill="none"
            stroke="#9ca3af"
            stroke-width="1"
            stroke-dasharray="4 3"
          />
        </svg>

        <!-- Hit layer: one transparent box per selectable element, in card
             units scaled to pixels. This is what Moveable and Selecto grab. -->
        <div
          v-for="el in hittable"
          :key="el.id"
          class="pc-hit absolute"
          :data-el-id="el.id"
          :style="{
            left: `${el.x * scale}px`,
            top: `${el.y * scale}px`,
            width: `${el.w * scale}px`,
            height: `${el.h * scale}px`,
            transform: el.rotation ? `rotate(${el.rotation}deg)` : undefined,
          }"
        />
      </div>

      <Moveable
        ref="moveableRef"
        :target="targets"
        :draggable="true"
        :resizable="true"
        :rotatable="true"
        :snappable="true"
        :origin="false"
        :keep-ratio="false"
        :throttle-drag="0"
        :throttle-resize="0"
        :throttle-rotate="0"
        :element-guidelines="guidelineTargets.map((el) => `[data-el-id='${el.id}']`)"
        :vertical-guidelines="verticalGuidelines"
        :horizontal-guidelines="horizontalGuidelines"
        :snap-directions="{ top: true, left: true, bottom: true, right: true, center: true, middle: true }"
        :element-snap-directions="{ top: true, left: true, bottom: true, right: true, center: true, middle: true }"
        @drag-start="onDragStart"
        @drag="onDrag"
        @drag-end="endGesture"
        @drag-group-end="endGesture"
        @resize-end="endGesture"
        @resize-group-end="endGesture"
        @rotate-end="endGesture"
        @rotate-group-end="endGesture"
        @drag-group-start="onDragGroupStart"
        @drag-group="onDragGroup"
        @resize-start="onResizeStart"
        @resize="onResize"
        @resize-group-start="onResizeGroupStart"
        @resize-group="onResizeGroup"
        @rotate-start="onRotateStart"
        @rotate="onRotate"
        @rotate-group-start="onRotateGroupStart"
        @rotate-group="onRotateGroup"
      />

      <Selecto
        ref="selectoRef"
        :drag-container="stageRef"
        :selectable-targets="['.pc-hit']"
        :hit-rate="0"
        :select-by-click="true"
        :select-from-inside="false"
        :toggle-continue-select="['shift']"
        @drag-start="onSelectoDragStart"
        @select-end="onSelectEnd"
      />
    </div>
  </div>
</template>

<style scoped>
.pc-hit {
  /* Transparent but hit-testable: the artwork underneath is what's seen. */
  background: transparent;
}
.pc-hit:hover {
  outline: 1px solid rgb(99 102 241 / 0.5);
}
</style>
