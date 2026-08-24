<script setup lang="ts">
withDefaults(defineProps<{ size?: number; color?: string }>(), { size: 18, color: 'currentColor' })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      :style="{ color }"
      fill="currentColor"
      d="M23 12s0-3.5-.45-5.15a2.9 2.9 0 0 0-2-2.05C18.9 4.3 12 4.3 12 4.3s-6.9 0-8.55.5a2.9 2.9 0 0 0-2 2.05C1 8.5 1 12 1 12s0 3.5.45 5.15a2.9 2.9 0 0 0 2 2.05C5.1 19.7 12 19.7 12 19.7s6.9 0 8.55-.5a2.9 2.9 0 0 0 2-2.05C23 15.5 23 12 23 12ZM9.75 15.4V8.6L15.8 12l-6.05 3.4Z"
    />
  </svg>
</template>
