<script setup lang="ts">
/**
 * The layers panel. It is a view onto the same element array the canvas
 * renders — reordering here changes paint order there, and vice versa,
 * because both read z-order from the store rather than keeping their own.
 *
 * Layers are listed top-first (the topmost element at the top of the list),
 * which is the opposite of the array's paint order, so every index crossing
 * that boundary is converted rather than passed through.
 */
import { computed } from 'vue'
import draggable from 'vuedraggable'
import { Copy, Eye, EyeOff, Lock, LockOpen, Trash2 } from '@lucide/vue'

import { useCardEditorStore } from '@/stores/cardEditor'
import { elementLabel, type CardElement } from '@/types/cardLayout'

const store = useCardEditorStore()

// Keyed loosely because vuedraggable's slot payload is untyped, so el.type
// arrives as `any` — a lookup that simply falls back is better here than
// casting the slot on every row.
const TYPE_ICON: Record<string, string> = {
  text: 'T',
  image: '▣',
  qr: '▦',
  shape: '◇',
  icon: '★',
  prompt: '⊙',
}

/**
 * vuedraggable needs a writable list. Writes come back as the reordered
 * top-first list, which is converted to a single move on the underlying
 * bottom-first array so the store keeps one undo entry per drag.
 */
const list = computed({
  get: () => store.layers,
  set: (next: CardElement[]) => {
    const total = next.length
    // Find the one element whose position changed and express it as a move.
    const before = store.layers
    let from = -1
    for (let i = 0; i < total; i += 1) {
      if (before[i]?.id !== next[i]?.id) {
        from = i
        break
      }
    }
    if (from === -1) return
    const moved = next[from]
    const oldIndex = before.findIndex((el) => el.id === moved.id)
    if (oldIndex === -1) return
    // Top-first index i corresponds to bottom-first index total-1-i.
    store.reorder(total - 1 - oldIndex, total - 1 - from)
  },
})

function onRowClick(el: CardElement, event: MouseEvent) {
  store.select(el.id, event.shiftKey || event.ctrlKey || event.metaKey)
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center justify-between border-b border-gray-200 px-3 py-2">
      <h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500">Capas</h3>
      <span class="text-xs text-gray-400">{{ store.elements.length }}</span>
    </div>

    <p v-if="store.elements.length === 0" class="px-3 py-6 text-center text-xs text-gray-400">
      Esta tarjeta no tiene elementos todavía.
    </p>

    <draggable
      v-else
      v-model="list"
      item-key="id"
      handle=".pc-layer-handle"
      class="flex-1 overflow-y-auto"
    >
      <template #item="{ element: el }">
        <div
          class="group flex items-center gap-1.5 border-b border-gray-100 px-2 py-2.5 text-sm"
          :class="[
            store.selectedIds.includes(el.id) ? 'bg-indigo-50' : 'hover:bg-gray-50',
            el.hidden ? 'opacity-50' : '',
          ]"
        >
          <span class="pc-layer-handle cursor-grab select-none px-1 text-gray-300" title="Arrastra para reordenar">⠿</span>

          <span class="w-4 shrink-0 text-center text-xs text-gray-400">{{ TYPE_ICON[el.type] }}</span>

          <button
            type="button"
            class="min-w-0 flex-1 truncate text-left"
            :class="el.locked ? 'text-gray-400' : 'text-gray-700'"
            :title="elementLabel(el)"
            @click="onRowClick(el, $event)"
          >
            {{ elementLabel(el) }}
          </button>

          <button
            type="button"
            class="rounded p-1 text-gray-400 opacity-0 transition group-hover:opacity-100 hover:bg-gray-200 hover:text-gray-700"
            :title="'Duplicar'"
            @click.stop="store.duplicate([el.id])"
          >
            <Copy :size="13" />
          </button>
          <button
            type="button"
            class="rounded p-1 hover:bg-gray-200"
            :class="el.locked ? 'text-indigo-600' : 'text-gray-400 opacity-0 group-hover:opacity-100'"
            :title="el.locked ? 'Desbloquear' : 'Bloquear'"
            @click.stop="store.toggleLocked(el.id)"
          >
            <component :is="el.locked ? Lock : LockOpen" :size="13" />
          </button>
          <button
            type="button"
            class="rounded p-1 hover:bg-gray-200"
            :class="el.hidden ? 'text-indigo-600' : 'text-gray-400 opacity-0 group-hover:opacity-100'"
            :title="el.hidden ? 'Mostrar' : 'Ocultar'"
            @click.stop="store.toggleHidden(el.id)"
          >
            <component :is="el.hidden ? EyeOff : Eye" :size="13" />
          </button>
          <button
            v-if="!el.locked"
            type="button"
            class="rounded p-1 text-gray-400 opacity-0 transition group-hover:opacity-100 hover:bg-red-100 hover:text-red-600"
            title="Eliminar"
            @click.stop="store.remove([el.id])"
          >
            <Trash2 :size="13" />
          </button>
        </div>
      </template>
    </draggable>
  </div>
</template>
