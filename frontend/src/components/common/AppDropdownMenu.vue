<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { MoreVertical } from '@lucide/vue'

const open = ref(false)
const rootEl = ref<HTMLElement | null>(null)
const panelStyle = ref({ top: '0px', left: '0px' })

function toggle() {
  if (open.value) {
    open.value = false
    return
  }
  const rect = rootEl.value?.getBoundingClientRect()
  if (rect) {
    panelStyle.value = { top: `${rect.bottom + 4}px`, left: `${rect.right - 208}px` }
  }
  open.value = true
}

function onClickOutside(e: MouseEvent) {
  if (rootEl.value && !rootEl.value.contains(e.target as Node)) {
    open.value = false
  }
}

function onScroll() {
  open.value = false
}

onMounted(() => {
  document.addEventListener('click', onClickOutside)
  window.addEventListener('scroll', onScroll, true)
})
onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  window.removeEventListener('scroll', onScroll, true)
})
</script>

<template>
  <div ref="rootEl" class="relative inline-block text-left">
    <button
      type="button"
      class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700"
      @click="toggle"
    >
      <MoreVertical :size="18" />
    </button>
    <Teleport to="body">
      <div
        v-if="open"
        class="fixed z-50 w-52 rounded-lg border border-gray-200 bg-white py-1 shadow-lg"
        :style="panelStyle"
        @click="open = false"
      >
        <slot />
      </div>
    </Teleport>
  </div>
</template>
