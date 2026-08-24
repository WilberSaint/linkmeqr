<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

defineProps<{ links: { to: string; label: string }[] }>()

const auth = useAuthStore()
const router = useRouter()

async function onLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="min-h-screen flex bg-gray-50">
    <aside class="w-56 shrink-0 border-r border-gray-200 bg-white flex flex-col">
      <div class="px-4 py-4 border-b border-gray-100">
        <p class="text-sm font-semibold text-gray-900">LinkMeQR</p>
        <p class="text-xs text-gray-500">{{ auth.user?.full_name }}</p>
      </div>
      <nav class="flex-1 px-2 py-3 space-y-0.5">
        <RouterLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="block rounded-lg px-3 py-2 text-sm text-gray-700 hover:bg-gray-100"
          active-class="bg-indigo-50 text-indigo-700 font-medium"
        >
          {{ link.label }}
        </RouterLink>
      </nav>
      <div class="p-2 border-t border-gray-100">
        <button
          class="w-full rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 text-left"
          @click="onLogout"
        >
          Cerrar sesión
        </button>
      </div>
    </aside>
    <main class="flex-1 min-w-0">
      <slot />
    </main>
  </div>
</template>
