<script setup lang="ts">
import { computed } from 'vue'

interface DayHours {
  day: string
  open: string
  close: string
  closed: boolean
}

const DAYS: { key: string; label: string }[] = [
  { key: 'mon', label: 'Lunes' },
  { key: 'tue', label: 'Martes' },
  { key: 'wed', label: 'Miércoles' },
  { key: 'thu', label: 'Jueves' },
  { key: 'fri', label: 'Viernes' },
  { key: 'sat', label: 'Sábado' },
  { key: 'sun', label: 'Domingo' },
]

const props = defineProps<{ content: Record<string, unknown> | null }>()
const emit = defineEmits<{ update: [content: Record<string, unknown>] }>()

const schedule = computed<DayHours[]>(() => {
  const raw = props.content?.schedule
  if (Array.isArray(raw) && raw.length === 7) return raw as DayHours[]
  return DAYS.map((d) => ({ day: d.key, open: '09:00', close: '18:00', closed: false }))
})

function updateDay(i: number, patch: Partial<DayHours>) {
  const next = schedule.value.map((row, idx) => (idx === i ? { ...row, ...patch } : row))
  emit('update', { schedule: next })
}
</script>

<template>
  <div class="space-y-1.5">
    <label class="block text-xs font-medium text-gray-600 mb-1">Horario</label>
    <div v-for="(row, i) in schedule" :key="row.day" class="flex items-center gap-1.5 text-xs">
      <span class="w-16 shrink-0 text-gray-600">{{ DAYS[i].label }}</span>
      <template v-if="!row.closed">
        <input
          type="time"
          :value="row.open"
          class="w-24 rounded border border-gray-300 px-1.5 py-1 text-xs"
          @change="updateDay(i, { open: ($event.target as HTMLInputElement).value })"
        />
        <span class="text-gray-400">–</span>
        <input
          type="time"
          :value="row.close"
          class="w-24 rounded border border-gray-300 px-1.5 py-1 text-xs"
          @change="updateDay(i, { close: ($event.target as HTMLInputElement).value })"
        />
      </template>
      <span v-else class="flex-1 text-gray-400 italic">Cerrado</span>
      <label class="ml-auto flex items-center gap-1 text-gray-500 shrink-0">
        <input type="checkbox" :checked="row.closed" @change="updateDay(i, { closed: ($event.target as HTMLInputElement).checked })" />
        Cerrado
      </label>
    </div>
  </div>
</template>
