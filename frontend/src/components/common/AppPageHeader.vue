<script setup lang="ts">
/**
 * The heading every admin and client screen opens with: a title, an optional
 * sentence saying what the screen is for, and a slot for its primary action.
 *
 * It exists because each screen had grown its own heading markup at its own
 * size, so moving between them felt like moving between different products.
 */
defineProps<{
  title: string
  /** One sentence on what this screen is for. Worth writing: most screens had none. */
  description?: string
}>()
</script>

<template>
  <header class="mb-6 flex flex-wrap items-start justify-between gap-4">
    <div class="min-w-0">
      <h1 class="text-xl font-semibold tracking-tight text-gray-900">{{ title }}</h1>
      <p v-if="description" class="mt-1 max-w-2xl text-sm text-gray-500">{{ description }}</p>
      <slot name="meta" />
    </div>
    <div class="flex shrink-0 items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
