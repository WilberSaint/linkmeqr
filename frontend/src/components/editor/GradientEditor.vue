<script setup lang="ts">
import { computed } from 'vue'
import { Pipette } from '@lucide/vue'
import { buildGradient, parseGradient } from '@/composables/gradientUtils'
import { isEyeDropperSupported, pickColorFromScreen } from '@/composables/useEyeDropper'

const props = defineProps<{ value: string }>()
const emit = defineEmits<{ update: [value: string] }>()

const parsed = computed(() => parseGradient(props.value) ?? { angle: 160, colorA: '#6366f1', colorB: '#a78bfa' })

function update(partial: Partial<{ angle: number; colorA: string; colorB: string }>) {
  const next = { ...parsed.value, ...partial }
  emit('update', buildGradient(next.angle, next.colorA, next.colorB))
}

async function pickWith(key: 'colorA' | 'colorB') {
  const color = await pickColorFromScreen()
  if (color) update({ [key]: color })
}
</script>

<template>
  <div class="space-y-3 p-3 bg-gray-50 rounded-lg border border-gray-200">
    <div class="flex items-center gap-3">
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1">Color 1</label>
        <div class="flex gap-1">
          <input
            type="color"
            :value="parsed.colorA"
            class="w-9 h-9 rounded-lg border border-gray-300"
            @input="update({ colorA: ($event.target as HTMLInputElement).value })"
          />
          <button
            v-if="isEyeDropperSupported"
            type="button"
            title="Elegir color de la pantalla"
            class="w-9 h-9 rounded-lg border border-gray-300 flex items-center justify-center text-gray-500 hover:border-indigo-400 hover:text-indigo-600"
            @click="pickWith('colorA')"
          >
            <Pipette :size="14" />
          </button>
        </div>
      </div>
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1">Color 2</label>
        <div class="flex gap-1">
          <input
            type="color"
            :value="parsed.colorB"
            class="w-9 h-9 rounded-lg border border-gray-300"
            @input="update({ colorB: ($event.target as HTMLInputElement).value })"
          />
          <button
            v-if="isEyeDropperSupported"
            type="button"
            title="Elegir color de la pantalla"
            class="w-9 h-9 rounded-lg border border-gray-300 flex items-center justify-center text-gray-500 hover:border-indigo-400 hover:text-indigo-600"
            @click="pickWith('colorB')"
          >
            <Pipette :size="14" />
          </button>
        </div>
      </div>
      <div
        class="flex-1 h-9 rounded-lg border border-gray-200 self-end"
        :style="{ background: buildGradient(parsed.angle, parsed.colorA, parsed.colorB) }"
      />
    </div>

    <div>
      <div class="flex items-center justify-between mb-1">
        <label class="text-[11px] font-medium text-gray-500">Ángulo</label>
        <span class="text-[11px] text-gray-400">{{ parsed.angle }}°</span>
      </div>
      <input
        type="range"
        min="0"
        max="360"
        :value="parsed.angle"
        class="w-full"
        @input="update({ angle: Number(($event.target as HTMLInputElement).value) })"
      />
    </div>
  </div>
</template>
