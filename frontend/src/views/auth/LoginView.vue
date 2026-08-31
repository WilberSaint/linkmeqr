<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { CircleAlert, Eye, EyeOff, Lock, Mail } from '@lucide/vue'
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
        <img
          src="/logo-blanco.png"
          alt="LinkMeQR"
          class="w-20 h-20 lg:w-28 lg:h-28 xl:w-32 xl:h-32 mb-6 lg:mb-10 drop-shadow-lg"
        />

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
          <img src="/logo-indigo.png" alt="LinkMeQR" class="w-12 h-12 shadow-sm shadow-indigo-200 mb-3 md:hidden" />
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
