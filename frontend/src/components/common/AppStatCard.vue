<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(
  defineProps<{
    label: string
    value: string | number
    icon?: Component
    tone?: 'indigo' | 'green' | 'amber' | 'red' | 'gray'
    hint?: string
  }>(),
  { tone: 'indigo' },
)
</script>

<template>
  <div class="bg-white border border-gray-200 rounded-xl p-4 flex items-start justify-between">
    <div class="min-w-0">
      <p class="text-xs font-medium text-gray-500">{{ label }}</p>
      <p class="mt-1 text-2xl font-semibold text-gray-900">{{ value }}</p>
      <p v-if="hint" class="mt-1 text-xs text-gray-400">{{ hint }}</p>
    </div>
    <div
      v-if="icon"
      class="rounded-lg p-2 shrink-0"
      :class="{
        'bg-indigo-50 text-indigo-600': tone === 'indigo',
        'bg-green-50 text-green-600': tone === 'green',
        'bg-amber-50 text-amber-600': tone === 'amber',
        'bg-red-50 text-red-600': tone === 'red',
        'bg-gray-50 text-gray-600': tone === 'gray',
      }"
    >
      <component :is="icon" :size="20" />
    </div>
  </div>
</template>
