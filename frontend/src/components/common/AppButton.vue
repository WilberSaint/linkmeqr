<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
    disabled?: boolean
    type?: 'button' | 'submit'
  }>(),
  { variant: 'primary', disabled: false, type: 'button' },
)
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    class="inline-flex items-center justify-center gap-1.5 rounded-lg px-3.5 py-2 text-sm font-medium transition disabled:opacity-50 disabled:cursor-not-allowed"
    :class="{
      'bg-indigo-600 text-white hover:bg-indigo-700': variant === 'primary',
      'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50': variant === 'secondary',
      'bg-red-600 text-white hover:bg-red-700': variant === 'danger',
      'text-gray-600 hover:bg-gray-100': variant === 'ghost',
    }"
  >
    <slot />
  </button>
</template>
