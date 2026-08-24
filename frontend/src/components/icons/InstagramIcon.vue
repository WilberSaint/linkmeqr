<script setup lang="ts">
withDefaults(defineProps<{ size?: number; color?: string }>(), { size: 18, color: 'currentColor' })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <rect x="2" y="2" width="20" height="20" rx="5" stroke="currentColor" :style="{ color }" stroke-width="2" />
    <circle cx="12" cy="12" r="4.2" stroke="currentColor" :style="{ color }" stroke-width="2" />
    <circle cx="17.2" cy="6.8" r="1.15" fill="currentColor" :style="{ color }" />
  </svg>
</template>
