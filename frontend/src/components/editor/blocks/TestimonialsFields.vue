<script setup lang="ts">
import { computed } from 'vue'
import { Trash2 } from '@lucide/vue'

interface Testimonial {
  author: string
  quote: string
  rating: number
}

const props = defineProps<{ content: Record<string, unknown> | null }>()
const emit = defineEmits<{ update: [content: Record<string, unknown>] }>()

const items = computed<Testimonial[]>(() => {
  const raw = props.content?.items
  return Array.isArray(raw) ? (raw as Testimonial[]) : []
})

function addItem() {
  emit('update', { items: [...items.value, { author: '', quote: '', rating: 5 }] })
}

function updateItem(i: number, patch: Partial<Testimonial>) {
  const next = items.value.map((it, idx) => (idx === i ? { ...it, ...patch } : it))
  emit('update', { items: next })
}

function removeItem(i: number) {
  emit('update', { items: items.value.filter((_, idx) => idx !== i) })
}
</script>

<template>
  <div class="space-y-2">
    <label class="block text-xs font-medium text-gray-600 mb-1">Testimonios</label>
    <div v-for="(t, i) in items" :key="i" class="rounded-lg border border-gray-200 p-2 space-y-1.5">
      <div class="flex items-center gap-1.5">
        <input
          :value="t.author"
          placeholder="Nombre"
          class="flex-1 rounded border border-gray-300 px-2 py-1 text-xs"
          @change="updateItem(i, { author: ($event.target as HTMLInputElement).value })"
        />
        <select
          :value="t.rating"
          class="rounded border border-gray-300 px-1 py-1 text-xs"
          @change="updateItem(i, { rating: Number(($event.target as HTMLSelectElement).value) })"
        >
          <option v-for="n in 5" :key="n" :value="n">{{ n }}★</option>
        </select>
        <button type="button" class="text-gray-400 hover:text-red-600 shrink-0" @click="removeItem(i)">
          <Trash2 :size="14" />
        </button>
      </div>
      <textarea
        :value="t.quote"
        placeholder="Cita del testimonio…"
        rows="2"
        class="w-full rounded border border-gray-300 px-2 py-1 text-xs"
        @change="updateItem(i, { quote: ($event.target as HTMLTextAreaElement).value })"
      />
    </div>
    <button type="button" class="text-xs text-indigo-600 hover:underline" @click="addItem">+ Agregar testimonio</button>
  </div>
</template>
