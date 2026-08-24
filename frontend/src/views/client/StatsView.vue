<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import * as statsApi from '@/api/stats'
import type { StatsSummary } from '@/api/stats'

const range = ref<'7d' | '30d'>('30d')
const summary = ref<StatsSummary | null>(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    summary.value = await statsApi.myStatsSummary(range.value)
  } finally {
    loading.value = false
  }
}

watch(range, load)
onMounted(load)
</script>

<template>
  <div class="p-6 max-w-4xl">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-lg font-semibold text-gray-900">Estadísticas</h1>
      <select v-model="range" class="rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm">
        <option value="7d">Últimos 7 días</option>
        <option value="30d">Últimos 30 días</option>
      </select>
    </div>

    <div v-if="summary" class="space-y-6">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <p class="text-xs text-gray-500">Visitas totales</p>
          <p class="text-xl font-semibold text-gray-900 mt-1">{{ summary.total_views }}</p>
        </div>
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <p class="text-xs text-gray-500">Clics en enlaces</p>
          <p class="text-xl font-semibold text-gray-900 mt-1">{{ summary.total_clicks }}</p>
        </div>
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <p class="text-xs text-gray-500">Últimos 7 días</p>
          <p class="text-xl font-semibold text-gray-900 mt-1">{{ summary.views_7d }}</p>
        </div>
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <p class="text-xs text-gray-500">Últimos 30 días</p>
          <p class="text-xl font-semibold text-gray-900 mt-1">{{ summary.views_30d }}</p>
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-xl p-5">
        <h2 class="text-sm font-semibold text-gray-900 mb-3">Visitas por día</h2>
        <div class="flex items-end gap-1 h-32">
          <div
            v-for="d in summary.timeseries"
            :key="d.date"
            class="flex-1 bg-indigo-500 rounded-t"
            :style="{ height: `${Math.max(4, (d.count / Math.max(...summary.timeseries.map((x) => x.count), 1)) * 100)}%` }"
            :title="`${d.date}: ${d.count}`"
          />
          <p v-if="summary.timeseries.length === 0" class="text-sm text-gray-400">Sin datos todavía.</p>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <h3 class="text-xs font-semibold text-gray-500 mb-2">Dispositivos</h3>
          <div v-for="d in summary.devices" :key="d.label" class="flex justify-between text-sm py-0.5">
            <span class="text-gray-600">{{ d.label }}</span>
            <span class="text-gray-900 font-medium">{{ d.count }}</span>
          </div>
        </div>
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <h3 class="text-xs font-semibold text-gray-500 mb-2">Sistema operativo</h3>
          <div v-for="d in summary.os" :key="d.label" class="flex justify-between text-sm py-0.5">
            <span class="text-gray-600">{{ d.label }}</span>
            <span class="text-gray-900 font-medium">{{ d.count }}</span>
          </div>
        </div>
        <div class="bg-white border border-gray-200 rounded-xl p-4">
          <h3 class="text-xs font-semibold text-gray-500 mb-2">Navegador</h3>
          <div v-for="d in summary.browsers" :key="d.label" class="flex justify-between text-sm py-0.5">
            <span class="text-gray-600">{{ d.label }}</span>
            <span class="text-gray-900 font-medium">{{ d.count }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
