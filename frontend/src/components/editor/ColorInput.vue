<script setup lang="ts">
import { Pipette } from '@lucide/vue'
import { isEyeDropperSupported, pickColorFromScreen } from '@/composables/useEyeDropper'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

async function onEyeDropper() {
  const color = await pickColorFromScreen()
  if (color) emit('update:modelValue', color)
}
</script>

<template>
  <div class="flex items-center gap-1.5">
    <input
      type="color"
      :value="modelValue"
      class="w-9 h-9 rounded-lg border border-gray-300 shrink-0"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <button
      v-if="isEyeDropperSupported"
      type="button"
      title="Elegir color de la pantalla (incluida la vista previa)"
      class="w-9 h-9 shrink-0 rounded-lg border border-gray-300 flex items-center justify-center text-gray-500 hover:border-indigo-400 hover:text-indigo-600 transition"
      @click="onEyeDropper"
    >
      <Pipette :size="16" />
    </button>
    <span class="text-xs text-gray-500 font-mono">{{ modelValue }}</span>
  </div>
</template>
