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

const DAY_ORDER = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']

function isToday(day: string) {
  const jsDay = new Date().getDay() // 0=Sun..6=Sat
  return DAY_ORDER[jsDay] === day
}

/** "HH:MM" → minutes since midnight, or null if it isn't a time at all. */
function toMinutes(hhmm: string | undefined): number | null {
  if (!hhmm) return null
  const m = /^(\d{1,2}):(\d{2})$/.exec(hhmm.trim())
  if (!m) return null
  const h = Number(m[1])
  const min = Number(m[2])
  if (h > 23 || min > 59) return null
  return h * 60 + min
}

/**
 * Whether the business is open at this exact moment — the one thing someone
 * scanning a QR outside the door actually wants to know.
 *
 * Returns null (rather than false) when it can't be determined — no schedule,
 * an unparseable time — so the badge is hidden instead of confidently
 * announcing "Cerrado" to a business that is, in fact, open.
 *
 * A closing time earlier than its opening time is read as running past
 * midnight (22:00–02:00), which is how bars and late-night kitchens set it.
 */
const openState = computed<{ open: boolean; until: string } | null>(() => {
  const rows = schedule.value
  if (rows.length === 0) return null

  const now = new Date()
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const todayKey = DAY_ORDER[now.getDay()]
  const yesterdayKey = DAY_ORDER[(now.getDay() + 6) % 7]

  const today = rows.find((r) => r.day === todayKey)
  if (today && !today.closed) {
    const open = toMinutes(today.open)
    const close = toMinutes(today.close)
    if (open !== null && close !== null) {
      if (close > open && nowMin >= open && nowMin < close) return { open: true, until: today.close }
      // Overnight span: still open if we're past opening time today.
      if (close < open && nowMin >= open) return { open: true, until: today.close }
    }
  }

  // Early morning hours belong to yesterday's overnight span, not today's.
  const yesterday = rows.find((r) => r.day === yesterdayKey)
  if (yesterday && !yesterday.closed) {
    const open = toMinutes(yesterday.open)
    const close = toMinutes(yesterday.close)
    if (open !== null && close !== null && close < open && nowMin < close) {
      return { open: true, until: yesterday.close }
    }
  }

  // Everything parsed, nothing matched — genuinely closed right now.
  const anyParsable = rows.some((r) => r.closed || (toMinutes(r.open) !== null && toMinutes(r.close) !== null))
  return anyParsable ? { open: false, until: '' } : null
})
</script>

<template>
  <div
    v-if="schedule.length"
    class="w-full rounded-xl px-4 py-3 text-sm space-y-1"
    :style="{ color: theme?.text_color, backgroundColor: 'rgba(0,0,0,0.04)' }"
  >
    <div v-if="openState" class="flex items-center gap-1.5 pb-1.5 mb-1 border-b border-current/10">
      <span
        class="w-2 h-2 rounded-full shrink-0"
        :style="{ backgroundColor: openState.open ? '#16a34a' : '#dc2626' }"
      />
      <span class="font-semibold">{{ openState.open ? 'Abierto ahora' : 'Cerrado ahora' }}</span>
      <span v-if="openState.open && openState.until" class="opacity-70">· cierra {{ openState.until }}</span>
    </div>
    <div v-for="row in schedule" :key="row.day" class="flex justify-between" :class="{ 'font-semibold': isToday(row.day) }">
      <span>{{ DAY_LABELS[row.day] ?? row.day }}</span>
      <span class="opacity-80">{{ row.closed ? 'Cerrado' : `${row.open} – ${row.close}` }}</span>
    </div>
  </div>
</template>
