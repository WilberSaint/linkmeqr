<script setup lang="ts">
import { computed } from 'vue'
import type { ProfileTheme } from '@/types'

interface DayHours {
  day: string
  open: string
  close: string
  closed: boolean
}

const props = defineProps<{ content: Record<string, unknown> | null; theme: ProfileTheme | null }>()

const DAY_LABELS: Record<string, string> = {
  mon: 'Lunes',
  tue: 'Martes',
  wed: 'Miércoles',
  thu: 'Jueves',
  fri: 'Viernes',
  sat: 'Sábado',
  sun: 'Domingo',
}

const schedule = computed<DayHours[]>(() => {
  const raw = props.content?.schedule
  return Array.isArray(raw) ? (raw as DayHours[]) : []
})

function isToday(day: string) {
  const jsDay = new Date().getDay() // 0=Sun..6=Sat
  const order = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
  return order[jsDay] === day
}
</script>

<template>
  <div
    v-if="schedule.length"
    class="w-full rounded-xl px-4 py-3 text-sm space-y-1"
    :style="{ color: theme?.text_color, backgroundColor: 'rgba(0,0,0,0.04)' }"
  >
    <div v-for="row in schedule" :key="row.day" class="flex justify-between" :class="{ 'font-semibold': isToday(row.day) }">
      <span>{{ DAY_LABELS[row.day] ?? row.day }}</span>
      <span class="opacity-80">{{ row.closed ? 'Cerrado' : `${row.open} – ${row.close}` }}</span>
    </div>
  </div>
</template>
