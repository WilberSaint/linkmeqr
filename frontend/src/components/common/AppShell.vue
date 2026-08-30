<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { LogOut, Menu, X } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'

defineProps<{ links: { to: string; label: string }[] }>()

const auth = useAuthStore()
const router = useRouter()
const mobileNavOpen = ref(false)

const initials = computed(() =>
  (auth.user?.full_name ?? '?')
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join(''),
)

async function onLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="min-h-screen flex bg-gray-50">
    <div class="md:hidden fixed top-0 inset-x-0 h-14 bg-white border-b border-gray-200 flex items-center justify-between px-3 z-30">
      <button class="p-2 text-gray-600" @click="mobileNavOpen = true">
        <Menu :size="20" />
      </button>
      <span class="text-sm font-semibold text-gray-900">LinkMeQR</span>
      <div class="w-8 h-8 rounded-full bg-indigo-100 text-indigo-700 text-xs font-semibold flex items-center justify-center shrink-0">
        {{ initials }}
      </div>
    </div>

    <div v-if="mobileNavOpen" class="md:hidden fixed inset-0 bg-black/40 z-40" @click="mobileNavOpen = false"></div>

    <aside
      class="w-64 md:w-56 shrink-0 border-r border-gray-200 bg-white flex flex-col fixed md:static inset-y-0 left-0 z-50 transition-transform duration-200 md:translate-x-0"
      :class="mobileNavOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <div class="px-4 py-4 border-b border-gray-100 flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-full bg-indigo-100 text-indigo-700 text-xs font-semibold flex items-center justify-center shrink-0">
          {{ initials }}
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-gray-900 truncate">{{ auth.user?.full_name }}</p>
          <p class="text-xs text-gray-400">LinkMeQR</p>
        </div>
        <button class="md:hidden p-1 text-gray-400 hover:text-gray-600 shrink-0" @click="mobileNavOpen = false">
          <X :size="18" />
        </button>
      </div>
      <nav class="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
        <RouterLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="block rounded-lg px-3 py-2 text-sm text-gray-700 hover:bg-gray-100"
          exact-active-class="bg-indigo-50 text-indigo-700 font-medium"
          @click="mobileNavOpen = false"
        >
          {{ link.label }}
        </RouterLink>
      </nav>
      <div class="p-2 border-t border-gray-100">
        <button
          class="w-full flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 text-left"
          @click="onLogout"
        >
          <LogOut :size="15" />
          Cerrar sesión
        </button>
      </div>
    </aside>
    <main class="flex-1 min-w-0 pt-14 md:pt-0">
      <slot />
    </main>
  </div>
</template>
