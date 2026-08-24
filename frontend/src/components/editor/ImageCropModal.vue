<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AppButton from '@/components/common/AppButton.vue'

type LogoShape = 'circle' | 'rounded' | 'square'

const props = defineProps<{ open: boolean; file: File | null; shape: LogoShape }>()
const emit = defineEmits<{ close: []; cropped: [blob: Blob, shape: LogoShape] }>()

const CANVAS_SIZE = 280

const canvasRef = ref<HTMLCanvasElement | null>(null)
const img = ref<HTMLImageElement | null>(null)
const zoom = ref(1)
const offsetX = ref(0)
const offsetY = ref(0)
const minZoom = ref(1)
const selectedShape = ref<LogoShape>(props.shape)

const dragging = ref(false)
const dragStart = ref({ x: 0, y: 0 })
const dragOffsetStart = ref({ x: 0, y: 0 })

const shapeOptions: { value: LogoShape; label: string; previewClass: string }[] = [
  { value: 'circle', label: 'Círculo', previewClass: 'rounded-full' },
  { value: 'rounded', label: 'Redondeado', previewClass: 'rounded-md' },
  { value: 'square', label: 'Cuadrado', previewClass: 'rounded-none' },
]

watch(
  () => props.open,
  (open) => {
    if (open) selectedShape.value = props.shape
  },
)

watch(
  () => props.file,
  (file) => {
    if (!file) return
    const url = URL.createObjectURL(file)
    const image = new Image()
    image.onload = () => {
      img.value = image
      const scale = Math.max(CANVAS_SIZE / image.width, CANVAS_SIZE / image.height)
      minZoom.value = scale
      zoom.value = scale
      offsetX.value = 0
      offsetY.value = 0
      draw()
    }
    image.src = url
  },
  { immediate: true },
)

watch([zoom, offsetX, offsetY, selectedShape], draw)

// A checkerboard pattern behind the image, like Photoshop/Figma transparency
// grids — without it, a white or transparent-background logo is invisible
// against the modal's own light background and you can't see what you're cropping.
function drawCheckerboard(ctx: CanvasRenderingContext2D) {
  const tile = 10
  for (let y = 0; y < CANVAS_SIZE; y += tile) {
    for (let x = 0; x < CANVAS_SIZE; x += tile) {
      const isEven = (Math.floor(x / tile) + Math.floor(y / tile)) % 2 === 0
      ctx.fillStyle = isEven ? '#e5e7eb' : '#f9fafb'
      ctx.fillRect(x, y, tile, tile)
    }
  }
}

function clipToShape(ctx: CanvasRenderingContext2D, shape: LogoShape) {
  ctx.beginPath()
  if (shape === 'circle') {
    ctx.arc(CANVAS_SIZE / 2, CANVAS_SIZE / 2, CANVAS_SIZE / 2, 0, Math.PI * 2)
  } else if (shape === 'rounded') {
    ctx.roundRect(0, 0, CANVAS_SIZE, CANVAS_SIZE, 24)
  } else {
    ctx.rect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
  }
}

function draw() {
  const canvas = canvasRef.value
  const image = img.value
  if (!canvas || !image) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  canvas.width = CANVAS_SIZE
  canvas.height = CANVAS_SIZE
  ctx.clearRect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
  drawCheckerboard(ctx)

  const w = image.width * zoom.value
  const h = image.height * zoom.value
  const x = (CANVAS_SIZE - w) / 2 + offsetX.value
  const y = (CANVAS_SIZE - h) / 2 + offsetY.value
  ctx.drawImage(image, x, y, w, h)

  // Darken everything outside the crop shape so the final crop area reads clearly.
  ctx.save()
  ctx.fillStyle = 'rgba(0,0,0,0.45)'
  ctx.fillRect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
  ctx.globalCompositeOperation = 'destination-out'
  clipToShape(ctx, selectedShape.value)
  ctx.fill()
  ctx.restore()

  // Crisp outline of the crop shape.
  ctx.save()
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 2
  clipToShape(ctx, selectedShape.value)
  ctx.stroke()
  ctx.restore()
}

function onPointerDown(e: PointerEvent) {
  dragging.value = true
  dragStart.value = { x: e.clientX, y: e.clientY }
  dragOffsetStart.value = { x: offsetX.value, y: offsetY.value }
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
}

function onPointerMove(e: PointerEvent) {
  if (!dragging.value) return
  offsetX.value = dragOffsetStart.value.x + (e.clientX - dragStart.value.x)
  offsetY.value = dragOffsetStart.value.y + (e.clientY - dragStart.value.y)
}

function onPointerUp() {
  dragging.value = false
}

const zoomPercent = computed({
  get: () => zoom.value,
  set: (v: number) => {
    zoom.value = v
  },
})

function onConfirm() {
  const canvas = canvasRef.value
  const image = img.value
  if (!canvas || !image) return

  // Re-render onto a clean output canvas without the darkened-overlay/outline
  // guides — those are only for the picker UI, not the exported image.
  const out = document.createElement('canvas')
  out.width = CANVAS_SIZE
  out.height = CANVAS_SIZE
  const ctx = out.getContext('2d')
  if (!ctx) return

  const w = image.width * zoom.value
  const h = image.height * zoom.value
  const x = (CANVAS_SIZE - w) / 2 + offsetX.value
  const y = (CANVAS_SIZE - h) / 2 + offsetY.value

  ctx.save()
  clipToShape(ctx, selectedShape.value)
  ctx.clip()
  ctx.drawImage(image, x, y, w, h)
  ctx.restore()

  out.toBlob((blob) => {
    if (blob) emit('cropped', blob, selectedShape.value)
  }, 'image/png')
}

onMounted(draw)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
      <div class="w-full max-w-sm rounded-xl bg-white p-5 shadow-xl">
        <h2 class="text-sm font-semibold text-gray-900 mb-3">Ajustar logo</h2>

        <div
          class="relative rounded-lg overflow-hidden touch-none select-none mx-auto"
          :style="{ width: `${CANVAS_SIZE}px`, height: `${CANVAS_SIZE}px` }"
        >
          <canvas
            ref="canvasRef"
            :width="CANVAS_SIZE"
            :height="CANVAS_SIZE"
            class="cursor-move"
            @pointerdown="onPointerDown"
            @pointermove="onPointerMove"
            @pointerup="onPointerUp"
            @pointerleave="onPointerUp"
          />
        </div>

        <div class="mt-4">
          <label class="block text-xs font-medium text-gray-600 mb-1.5">Forma</label>
          <div class="flex gap-2">
            <button
              v-for="s in shapeOptions"
              :key="s.value"
              type="button"
              class="flex-1 flex flex-col items-center gap-1 rounded-lg border-2 py-2 transition"
              :class="selectedShape === s.value ? 'border-indigo-500 bg-indigo-50' : 'border-gray-200 hover:border-gray-300'"
              @click="selectedShape = s.value"
            >
              <span class="w-7 h-7 bg-gray-400" :class="s.previewClass" />
              <span class="text-[11px] text-gray-600">{{ s.label }}</span>
            </button>
          </div>
        </div>

        <div class="mt-4">
          <label class="block text-xs font-medium text-gray-600 mb-1">Zoom</label>
          <input
            v-model.number="zoomPercent"
            type="range"
            :min="minZoom"
            :max="minZoom * 4"
            :step="minZoom / 50"
            class="w-full"
          />
        </div>
        <p class="text-xs text-gray-400 mt-1">Arrastra la imagen para reposicionarla.</p>

        <div class="flex justify-end gap-2 mt-4">
          <AppButton variant="secondary" @click="emit('close')">Cancelar</AppButton>
          <AppButton @click="onConfirm">Usar esta imagen</AppButton>
        </div>
      </div>
    </div>
  </Teleport>
</template>
