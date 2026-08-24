<script setup lang="ts">
withDefaults(
  defineProps<{ tone?: 'green' | 'red' | 'gray' | 'amber' | 'indigo' }>(),
  { tone: 'gray' },
)
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
    :class="{
      'bg-green-100 text-green-700': tone === 'green',
      'bg-red-100 text-red-700': tone === 'red',
      'bg-gray-100 text-gray-700': tone === 'gray',
      'bg-amber-100 text-amber-700': tone === 'amber',
      'bg-indigo-100 text-indigo-700': tone === 'indigo',
    }"
  >
    <slot />
  </span>
</template>
