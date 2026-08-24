<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Template } from '@/types'
import * as templatesApi from '@/api/templates'

const templates = ref<Template[]>([])

onMounted(async () => {
  templates.value = await templatesApi.listTemplates()
})
</script>

<template>
  <div class="p-6 max-w-4xl">
    <h1 class="text-lg font-semibold text-gray-900 mb-6">Plantillas</h1>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div
        v-for="t in templates"
        :key="t.id"
        class="border border-gray-200 rounded-xl p-4 bg-white"
        :style="{
          background: t.default_theme.background_type === 'gradient' ? t.default_theme.background_value : undefined,
        }"
      >
        <p class="text-sm font-semibold text-gray-900">{{ t.name }}</p>
        <p class="text-xs text-gray-500 mt-1">{{ t.description }}</p>
      </div>
    </div>
  </div>
</template>
