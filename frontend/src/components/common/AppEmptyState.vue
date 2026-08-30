<script setup lang="ts">
/**
 * What a screen shows before it has any data.
 *
 * Every list here used to render a bare "Sin registros todavía." — which tells
 * someone that nothing is there, but not how to change that. An empty screen
 * is the moment a person is most likely to be lost, so this one takes the
 * space to explain what the thing is and offer the action that creates the
 * first one.
 */
defineProps<{
  title: string
  description?: string
  icon?: unknown
}>()
</script>

<template>
  <div class="flex flex-col items-center px-6 py-14 text-center">
    <div v-if="icon" class="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-400">
      <component :is="icon" :size="22" />
    </div>
    <p class="text-sm font-medium text-gray-900">{{ title }}</p>
    <p v-if="description" class="mt-1.5 max-w-sm text-sm text-gray-500">{{ description }}</p>
    <div v-if="$slots.action" class="mt-5">
      <slot name="action" />
    </div>
  </div>
</template>
