<script setup lang="ts">
defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-8">
      <div class="w-full max-w-md max-h-full rounded-xl bg-white shadow-xl flex flex-col overflow-hidden">
        <div class="flex items-center justify-between px-6 pt-6 pb-4 shrink-0">
          <h2 class="text-base font-semibold text-gray-900">{{ title }}</h2>
          <button class="text-gray-400 hover:text-gray-600" @click="emit('close')">✕</button>
        </div>
        <div class="px-6 pb-6 overflow-y-auto flex-1 min-h-0">
          <slot />
        </div>
        <div v-if="$slots.footer" class="shrink-0 border-t border-gray-100 px-6 py-4 flex justify-end gap-2">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>
