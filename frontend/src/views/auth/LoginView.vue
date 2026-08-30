<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { CircleAlert, Eye, EyeOff, Lock, Mail, QrCode } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value)
    if (auth.isAdmin) {
      router.push({ name: 'admin-dashboard' })
    } else {
      router.push({ name: 'client-dashboard' })
    }
  } catch {
    error.value = 'Credenciales inválidas.'
  } finally {
    loading.value = false
  }
}

// A decorative, non-scannable QR-art pattern for the brand panel — a 9x9
// module grid with the 3 finder squares a real QR always has (so it reads
// unmistakably as "a QR" at a glance) and a deterministic pseudo-random
// scatter of filled modules everywhere else, computed once from a fixed
// seed so the pattern is stable across renders instead of reshuffling.
const qrArtModules = computed(() => {
  const size = 9
  const isFinderZone = (x: number, y: number) => (x < 3 && y < 3) || (x > size - 4 && y < 3) || (x < 3 && y > size - 4)
  let seed = 42
  const rand = () => {
    seed = (seed * 1103515245 + 12345) & 0x7fffffff
    return (seed % 1000) / 1000
  }
  const cells: { x: number; y: number }[] = []
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      if (isFinderZone(x, y)) continue
      if (rand() > 0.55) cells.push({ x, y })
    }
  }
  return cells
})
</script>

<template>
  <div class="min-h-screen flex bg-white">
    <!-- Brand panel: hidden on phones, the visual "llamativo" half from md+
         (kicks in at tablet width, not just laptop, so the awkward
         "small card floating in a mostly-empty screen" look doesn't
         persist all the way through every tablet size). Content scales up
         across md/lg/xl so it neither feels cramped at 768px nor lost in
         empty space on a big desktop monitor. -->
    <div class="hidden md:flex md:w-2/5 lg:w-1/2 relative overflow-hidden bg-gradient-to-br from-indigo-600 via-indigo-700 to-violet-800 items-center justify-center p-8 lg:p-12">
      <div class="absolute -top-24 -left-24 w-96 h-96 rounded-full bg-white/10 blur-3xl"></div>
      <div class="absolute -bottom-32 -right-16 w-96 h-96 rounded-full bg-fuchsia-400/20 blur-3xl"></div>

      <div class="relative z-10 max-w-xs lg:max-w-sm xl:max-w-md text-white">
        <svg viewBox="0 0 9 9" class="w-28 h-28 lg:w-40 lg:h-40 xl:w-48 xl:h-48 mb-6 lg:mb-10 drop-shadow-lg">
          <rect x="0" y="0" width="3" height="3" rx="0.5" fill="none" stroke="white" stroke-width="0.5" opacity="0.9" />
          <rect x="1" y="1" width="1" height="1" fill="white" opacity="0.9" />
          <rect x="6" y="0" width="3" height="3" rx="0.5" fill="none" stroke="white" stroke-width="0.5" opacity="0.9" />
          <rect x="7" y="1" width="1" height="1" fill="white" opacity="0.9" />
          <rect x="0" y="6" width="3" height="3" rx="0.5" fill="none" stroke="white" stroke-width="0.5" opacity="0.9" />
          <rect x="1" y="7" width="1" height="1" fill="white" opacity="0.9" />
          <rect
            v-for="(c, i) in qrArtModules"
            :key="i"
            :x="c.x"
            :y="c.y"
            width="1"
            height="1"
            rx="0.18"
            fill="white"
            :opacity="0.25 + ((c.x * 7 + c.y * 13) % 5) * 0.1"
          />
        </svg>

        <h1 class="text-2xl lg:text-3xl xl:text-4xl font-bold leading-tight mb-3">Todo tu negocio,<br />en un solo QR</h1>
        <p class="text-indigo-100 text-sm xl:text-base leading-relaxed">
          Perfil, reseñas, lealtad y tarjetas para imprimir — todo lo que tu negocio necesita para convertir una
          mirada en un cliente.
        </p>
      </div>
    </div>

    <!-- Form panel -->
    <div class="flex-1 flex items-center justify-center px-4 py-12 bg-gradient-to-br from-indigo-50 via-white to-white md:bg-none">
      <div class="w-full max-w-sm">
        <div class="flex flex-col items-center md:items-start mb-6">
          <div class="w-12 h-12 rounded-2xl bg-indigo-600 text-white flex items-center justify-center shadow-sm shadow-indigo-200 mb-3 md:hidden">
            <QrCode :size="24" />
          </div>
          <h1 class="text-xl font-semibold text-gray-900">LinkMeQR</h1>
          <p class="text-sm text-gray-500 mt-0.5">Todo tu negocio en un QR</p>
        </div>

        <div class="bg-white shadow-sm rounded-xl p-8 border border-gray-100">
          <form class="space-y-4" @submit.prevent="onSubmit">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <div class="relative">
                <Mail :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
                <input
                  v-model="email"
                  type="email"
                  required
                  autocomplete="email"
                  class="w-full rounded-lg border border-gray-300 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Contraseña</label>
              <div class="relative">
                <Lock :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
                <input
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  required
                  autocomplete="current-password"
                  class="w-full rounded-lg border border-gray-300 pl-9 pr-9 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
                <button
                  type="button"
                  tabindex="-1"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  :aria-label="showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'"
                  @click="showPassword = !showPassword"
                >
                  <EyeOff v-if="showPassword" :size="16" />
                  <Eye v-else :size="16" />
                </button>
              </div>
            </div>

            <p v-if="error" class="flex items-center gap-1.5 text-sm text-red-700 bg-red-50 border border-red-100 rounded-lg px-3 py-2">
              <CircleAlert :size="15" class="shrink-0" />
              {{ error }}
            </p>

            <button
              type="submit"
              :disabled="loading"
              class="w-full rounded-lg bg-indigo-600 text-white text-sm font-medium py-2.5 hover:bg-indigo-700 transition disabled:opacity-50"
            >
              {{ loading ? 'Ingresando…' : 'Iniciar sesión' }}
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>
