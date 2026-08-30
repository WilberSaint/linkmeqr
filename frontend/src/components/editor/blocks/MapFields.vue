<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ content: Record<string, unknown> | null }>()
const emit = defineEmits<{ update: [content: Record<string, unknown>] }>()

const embedUrl = computed(() => {
  const url = props.content?.embed_url
  return typeof url === 'string' ? url : ''
})
</script>

<template>
  <div>
    <label class="block text-xs font-medium text-gray-600 mb-1">Link de mapa insertado</label>
    <input
      :value="embedUrl"
      placeholder="https://www.google.com/maps/embed?..."
      class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm"
      @change="emit('update', { embed_url: ($event.target as HTMLInputElement).value })"
    />
    <p class="text-[11px] text-gray-400 mt-1">
      En Google Maps: busca tu negocio → Compartir → pestaña "Insertar un mapa" → copia el link que está
      dentro de <code>src="…"</code> y pégalo aquí.
    </p>
  </div>
</template>
