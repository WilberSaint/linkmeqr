import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import {
  defaultProps,
  defaultSize,
  type CardElement,
  type CardLayout,
  type ElementProps,
  type ElementType,
  type PromptVariant,
} from '@/types/cardLayout'

/**
 * The print-card designer's state. Every mutation in the editor goes through
 * here — the canvas, the layers panel and the properties panel are all views
 * onto this one tree, which is why dragging an element updates its layer row
 * and its property fields without any of them knowing about each other.
 */

const HISTORY_LIMIT = 60

function newId(): string {
  // crypto.randomUUID is available in every browser this app supports; the
  // fallback only matters for insecure-origin dev servers, where it is still
  // unique enough for element ids that never leave one document.
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `el-${Math.random().toString(36).slice(2, 10)}`
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export const useCardEditorStore = defineStore('cardEditor', () => {
  const layout = ref<CardLayout | null>(null)
  const selectedIds = ref<string[]>([])
  /** The revision the tree was loaded at, sent back on save to detect a concurrent edit. */
  const baseVersion = ref<number | null>(null)
  const dirty = ref(false)

  const undoStack = ref<CardLayout[]>([])
  const redoStack = ref<CardLayout[]>([])

  const elements = computed(() => layout.value?.elements ?? [])

  /** Layers are listed top-first, the way every design tool shows them. */
  const layers = computed(() => [...elements.value].slice().reverse())

  const selected = computed(() => elements.value.filter((el) => selectedIds.value.includes(el.id)))

  /** The properties panel edits a single element; a multi-selection has no single set of props. */
  const singleSelected = computed<CardElement | null>(() => (selected.value.length === 1 ? selected.value[0] : null))

  const canUndo = computed(() => undoStack.value.length > 0)
  const canRedo = computed(() => redoStack.value.length > 0)

  function load(next: CardLayout, version: number) {
    layout.value = clone(next)
    baseVersion.value = version
    selectedIds.value = []
    undoStack.value = []
    redoStack.value = []
    dirty.value = false
    normalizeZ()
  }

  /**
   * Snapshot the tree before a mutation. Call it once per user-visible
   * action, not per frame: a drag pushes one entry on pointer-down, so undo
   * steps back over the whole drag rather than one pixel of it.
   */
  function snapshot() {
    if (!layout.value) return
    undoStack.value.push(clone(layout.value))
    if (undoStack.value.length > HISTORY_LIMIT) undoStack.value.shift()
    redoStack.value = []
    dirty.value = true
  }

  function undo() {
    if (!layout.value || undoStack.value.length === 0) return
    redoStack.value.push(clone(layout.value))
    layout.value = undoStack.value.pop()!
    pruneSelection()
    dirty.value = true
  }

  function redo() {
    if (!layout.value || redoStack.value.length === 0) return
    undoStack.value.push(clone(layout.value))
    layout.value = redoStack.value.pop()!
    pruneSelection()
    dirty.value = true
  }

  /** Drop ids that no longer exist, so an undo past a delete leaves no ghost selection. */
  function pruneSelection() {
    const present = new Set(elements.value.map((el) => el.id))
    selectedIds.value = selectedIds.value.filter((id) => present.has(id))
  }

  function find(id: string): CardElement | undefined {
    return layout.value?.elements.find((el) => el.id === id)
  }

  // --- selection ----------------------------------------------------------

  function select(id: string, additive = false) {
    const el = find(id)
    // A locked element stays visible and rendered but cannot be picked up —
    // that is the entire point of locking it.
    if (!el || el.locked) return
    if (additive) {
      selectedIds.value = selectedIds.value.includes(id)
        ? selectedIds.value.filter((x) => x !== id)
        : [...selectedIds.value, id]
    } else {
      selectedIds.value = [id]
    }
  }

  function selectMany(ids: string[]) {
    const selectable = new Set(elements.value.filter((el) => !el.locked && !el.hidden).map((el) => el.id))
    selectedIds.value = ids.filter((id) => selectable.has(id))
  }

  function clearSelection() {
    selectedIds.value = []
  }

  // --- geometry -----------------------------------------------------------

  /**
   * Apply a geometry delta to one element. Called continuously while dragging
   * or resizing, so it deliberately does NOT snapshot — the caller pushes one
   * history entry when the gesture starts.
   */
  function setGeometry(id: string, geo: Partial<Pick<CardElement, 'x' | 'y' | 'w' | 'h' | 'rotation'>>) {
    const el = find(id)
    if (!el || el.locked) return
    if (geo.x !== undefined) el.x = geo.x
    if (geo.y !== undefined) el.y = geo.y
    // A zero or negative box would make the element unselectable and
    // invisible — clamp instead of letting a resize handle destroy it.
    if (geo.w !== undefined) el.w = Math.max(1, geo.w)
    if (geo.h !== undefined) el.h = Math.max(1, geo.h)
    // Moveable reports rotation as an ever-accumulating delta (spin the
    // handle past a full turn and it just keeps counting up or down), but
    // nothing about the element itself cares which lap it's on — wrap it
    // back into [0, 360) so the number (and the input in the properties
    // panel) doesn't climb without bound.
    if (geo.rotation !== undefined) el.rotation = ((geo.rotation % 360) + 360) % 360
    dirty.value = true
  }

  function nudge(dx: number, dy: number) {
    if (selected.value.length === 0) return
    snapshot()
    for (const el of selected.value) {
      if (el.locked) continue
      el.x += dx
      el.y += dy
    }
  }

  // --- properties ---------------------------------------------------------

  function setProps(id: string, patch: ElementProps) {
    const el = find(id)
    if (!el) return
    snapshot()
    el.props = { ...el.props, ...patch }
  }

  function setElement(id: string, patch: Partial<CardElement>) {
    const el = find(id)
    if (!el) return
    snapshot()
    // Same wrap as setGeometry — this is the path the properties panel's own
    // rotation number field writes through, and typing e.g. 400 should land
    // on 40, not persist as-is.
    if (patch.rotation !== undefined) patch = { ...patch, rotation: ((patch.rotation % 360) + 360) % 360 }
    Object.assign(el, patch)
  }

  function setBackground(patch: Partial<CardLayout['background']>) {
    if (!layout.value) return
    snapshot()
    layout.value.background = { ...layout.value.background, ...patch }
  }

  /**
   * Edits the printable area itself. Deliberately narrow: only the corner
   * radius is offered, because changing w/h here would move every element
   * relative to the card without touching their coordinates — a resize has to
   * go through the card record and re-seed instead.
   */
  function setCanvas(patch: Partial<Pick<CardLayout['canvas'], 'corner_r'>>) {
    if (!layout.value) return
    snapshot()
    layout.value.canvas = { ...layout.value.canvas, ...patch }
  }

  // --- adding and removing ------------------------------------------------

  function addElement(type: ElementType, at?: { x: number; y: number }, variant?: PromptVariant): string | null {
    if (!layout.value) return null
    snapshot()
    const size = defaultSize(type)
    const canvas = layout.value.canvas
    const el: CardElement = {
      id: newId(),
      type,
      // Default to the middle of the card so a new element always lands
      // somewhere visible, whatever the canvas is scrolled or zoomed to.
      x: at?.x ?? (canvas.w - size.w) / 2,
      y: at?.y ?? (canvas.h - size.h) / 2,
      w: size.w,
      h: size.h,
      rotation: 0,
      z_index: layout.value.elements.length,
      hidden: false,
      locked: false,
      props: defaultProps(type, variant),
    }
    layout.value.elements.push(el)
    normalizeZ()
    selectedIds.value = [el.id]
    return el.id
  }

  /**
   * Adds a QR plus a "Toca" and "Escanea" prompt pair, centred under it —
   * the toolbar's "+ QR" action, so a fresh QR starts with the same
   * call-to-action a seeded card would, except as three independent
   * elements from the first moment: move "Escanea" to the QR's right, drop
   * "Toca" entirely, restyle either on its own, all without touching the
   * QR itself.
   */
  function addQRWithPrompts(at?: { x: number; y: number }): string | null {
    if (!layout.value) return null
    snapshot()
    const canvas = layout.value.canvas
    const qrSize = defaultSize('qr')
    const qrX = at?.x ?? (canvas.w - qrSize.w) / 2
    const qrY = at?.y ?? (canvas.h - qrSize.h) / 2

    const qrEl: CardElement = {
      id: newId(),
      type: 'qr',
      x: qrX,
      y: qrY,
      w: qrSize.w,
      h: qrSize.h,
      rotation: 0,
      z_index: layout.value.elements.length,
      hidden: false,
      locked: false,
      props: defaultProps('qr'),
    }
    layout.value.elements.push(qrEl)

    // A rough width estimate — good enough to size the two boxes so their
    // content fits without a visible resize on first render; the designer
    // can resize either freely afterward.
    const fontSize = 8
    const estWidth = (text: string) => text.length * fontSize * 0.56 + fontSize * 1.05 + fontSize * 0.3
    const gap = fontSize
    const tapW = estWidth('Toca')
    const scanW = estWidth('Escanea')
    const promptH = fontSize * 1.4
    const cx = qrX + qrSize.w / 2
    const rowY = qrY + qrSize.h + 8
    const startX = cx - (tapW + gap + scanW) / 2

    const tapEl: CardElement = {
      id: newId(),
      type: 'prompt',
      x: startX,
      y: rowY,
      w: tapW,
      h: promptH,
      rotation: 0,
      z_index: layout.value.elements.length,
      hidden: false,
      locked: false,
      props: defaultProps('prompt', 'tap'),
    }
    const scanEl: CardElement = {
      id: newId(),
      type: 'prompt',
      x: startX + tapW + gap,
      y: rowY,
      w: scanW,
      h: promptH,
      rotation: 0,
      z_index: layout.value.elements.length + 1,
      hidden: false,
      locked: false,
      props: defaultProps('prompt', 'scan'),
    }
    layout.value.elements.push(tapEl, scanEl)

    normalizeZ()
    selectedIds.value = [qrEl.id]
    return qrEl.id
  }

  function duplicate(ids: string[] = selectedIds.value) {
    if (!layout.value || ids.length === 0) return
    snapshot()
    const copies: CardElement[] = []
    for (const id of ids) {
      const el = find(id)
      if (!el) continue
      const copy = clone(el)
      copy.id = newId()
      // Offset slightly so the copy is visibly on top of the original rather
      // than perfectly hidden behind it.
      copy.x += 8
      copy.y += 8
      copy.z_index = layout.value.elements.length + copies.length
      // A duplicated QR must not inherit the original's printed URL segment,
      // or both would resolve to the same scan slot and the new one would
      // silently steal the old card's analytics.
      if (copy.type === 'qr') delete copy.props.legacy_slot
      copies.push(copy)
    }
    layout.value.elements.push(...copies)
    normalizeZ()
    selectedIds.value = copies.map((c) => c.id)
  }

  function remove(ids: string[] = selectedIds.value) {
    if (!layout.value || ids.length === 0) return
    // A locked element is protected from deletion, so filter those out FIRST:
    // snapshotting for a delete that removes nothing would mark the document
    // dirty and push a pointless undo step.
    const doomed = new Set(ids.filter((id) => find(id)?.locked === false))
    if (doomed.size === 0) return
    snapshot()
    layout.value.elements = layout.value.elements.filter((el) => !doomed.has(el.id))
    normalizeZ()
    pruneSelection()
  }

  // --- layers -------------------------------------------------------------

  function toggleHidden(id: string) {
    const el = find(id)
    if (!el) return
    snapshot()
    el.hidden = !el.hidden
    // A hidden element cannot stay selected: its handles would float over
    // nothing and a drag would move something invisible.
    if (el.hidden) selectedIds.value = selectedIds.value.filter((x) => x !== id)
  }

  function toggleLocked(id: string) {
    const el = find(id)
    if (!el) return
    snapshot()
    el.locked = !el.locked
    if (el.locked) selectedIds.value = selectedIds.value.filter((x) => x !== id)
  }

  /**
   * Move an element to a new position in paint order. `to` is an index into
   * the bottom-first elements array, which is what reorder callers convert
   * the (top-first) layers list into.
   */
  function reorder(from: number, to: number) {
    if (!layout.value) return
    if (from === to || from < 0 || to < 0) return
    if (from >= layout.value.elements.length || to >= layout.value.elements.length) return
    snapshot()
    const [moved] = layout.value.elements.splice(from, 1)
    layout.value.elements.splice(to, 0, moved)
    normalizeZ()
  }

  function bringToFront(id: string) {
    const idx = elements.value.findIndex((el) => el.id === id)
    if (idx >= 0) reorder(idx, elements.value.length - 1)
  }

  function sendToBack(id: string) {
    const idx = elements.value.findIndex((el) => el.id === id)
    if (idx >= 0) reorder(idx, 0)
  }

  /**
   * Rewrite z_index to a dense 0..n-1 run matching array order. The backend
   * does the same on save; doing it here too means the editor's own ordering
   * never disagrees with what a reload would show.
   */
  function normalizeZ() {
    if (!layout.value) return
    layout.value.elements.forEach((el, i) => {
      el.z_index = i
    })
  }

  function markSaved(version: number) {
    baseVersion.value = version
    dirty.value = false
  }

  return {
    layout,
    elements,
    layers,
    selectedIds,
    selected,
    singleSelected,
    baseVersion,
    dirty,
    canUndo,
    canRedo,
    load,
    snapshot,
    undo,
    redo,
    find,
    select,
    selectMany,
    clearSelection,
    setGeometry,
    nudge,
    setProps,
    setElement,
    setBackground,
    setCanvas,
    addElement,
    addQRWithPrompts,
    duplicate,
    remove,
    toggleHidden,
    toggleLocked,
    reorder,
    bringToFront,
    sendToBack,
    markSaved,
  }
})
