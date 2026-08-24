<script setup lang="ts">
import { computed } from 'vue'
import { PATTERNS, buildPatternBackground, parsePatternBackground, type PatternDirection } from '@/composables/backgroundPatterns'
import ColorInput from './ColorInput.vue'

const props = defineProps<{ value: string }>()
const emit = defineEmits<{ update: [value: string] }>()

const parsed = computed(
  () => parsePatternBackground(props.value) ?? { patternId: PATTERNS[0].id, baseColor: '#ffffff', motifColor: '#111827', direction: 'horizontal' as PatternDirection, spacing: 40 },
)

function update(partial: Partial<{ patternId: string; baseColor: string; motifColor: string; direction: PatternDirection; spacing: number }>) {
  const next = { ...parsed.value, ...partial }
  emit('update', buildPatternBackground(next.patternId, next.baseColor, next.motifColor, next.direction, next.spacing))
}

const directions: { value: PatternDirection; label: string }[] = [
  { value: 'horizontal', label: 'Horizontal' },
  { value: 'vertical', label: 'Vertical' },
  { value: 'diagonal', label: 'Diagonal' },
]
</script>

<template>
  <div class="space-y-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
    <div>
      <label class="block text-[11px] font-medium text-gray-500 mb-1.5">Motivo</label>
      <div class="grid grid-cols-4 gap-1.5">
        <button
          v-for="p in PATTERNS"
          :key="p.id"
          type="button"
          class="aspect-square rounded-md border-2 flex items-center justify-center transition"
          :class="parsed.patternId === p.id ? 'border-indigo-500 bg-indigo-50' : 'border-gray-200 hover:border-gray-300 bg-white'"
          :title="p.label"
          @click="update({ patternId: p.id })"
        >
          <div
            class="w-full h-full"
            :style="{
              background: buildPatternBackground(p.id, '#ffffff', '#374151', 'horizontal', 16),
              backgroundSize: '16px 16px',
            }"
          />
        </button>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1">Color de fondo</label>
        <ColorInput :model-value="parsed.baseColor" @update:model-value="(v) => update({ baseColor: v })" />
      </div>
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1">Color del motivo</label>
        <ColorInput :model-value="parsed.motifColor" @update:model-value="(v) => update({ motifColor: v })" />
      </div>
    </div>

    <div>
      <label class="block text-[11px] font-medium text-gray-500 mb-1.5">Dirección</label>
      <div class="flex gap-1.5">
        <button
          v-for="d in directions"
          :key="d.value"
          type="button"
          class="flex-1 rounded-md border-2 py-1.5 text-[11px] font-medium transition"
          :class="parsed.direction === d.value ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-500 hover:border-gray-300'"
          @click="update({ direction: d.value })"
        >
          {{ d.label }}
        </button>
      </div>
    </div>

    <div>
      <div class="flex items-center justify-between mb-1">
        <label class="text-[11px] font-medium text-gray-500">Tamaño</label>
        <span class="text-[11px] text-gray-400">{{ parsed.spacing }}px</span>
      </div>
      <input
        type="range"
        min="20"
        max="80"
        :value="parsed.spacing"
        class="w-full"
        @input="update({ spacing: Number(($event.target as HTMLInputElement).value) })"
      />
    </div>

    <div
      class="h-16 rounded-lg border border-gray-200"
      :style="{
        background: buildPatternBackground(parsed.patternId, parsed.baseColor, parsed.motifColor, parsed.direction, parsed.spacing),
      }"
    />
  </div>
</template>
