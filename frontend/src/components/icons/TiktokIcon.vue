<script setup lang="ts">
withDefaults(defineProps<{ size?: number; color?: string }>(), { size: 18, color: 'currentColor' })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      :style="{ color }"
      fill="currentColor"
      d="M16.6 2h-3.2v13.6a2.8 2.8 0 1 1-2.8-2.8c.24 0 .48.02.7.07V9.6a5.9 5.9 0 0 0-.7-.04A5.99 5.99 0 1 0 16.6 15.6V8.3a8.1 8.1 0 0 0 4.4 1.3V6.4a4.9 4.9 0 0 1-4.4-4.4Z"
    />
  </svg>
</template>
