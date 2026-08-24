<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useLicenseStore } from '@/stores/license'
import AppBadge from '@/components/common/AppBadge.vue'

const auth = useAuthStore()
const licenseStore = useLicenseStore()

onMounted(() => licenseStore.refresh())

function statusTone(status?: string) {
  if (status === 'ACTIVE') return 'green'
  if (status === 'EXPIRED') return 'red'
  return 'gray'
}
</script>

<template>
  <div class="p-6 max-w-3xl space-y-6">
    <div>
      <h1 class="text-lg font-semibold text-gray-900">Bienvenido, {{ auth.user?.full_name }}</h1>
      <p class="text-sm text-gray-500 mt-1">Panel de cliente — LinkMeQR</p>
    </div>

    <div class="bg-white border border-gray-200 rounded-xl p-5 flex items-center justify-between">
      <div>
        <p class="text-sm text-gray-500">Estado de tu licencia</p>
        <p class="text-sm text-gray-900 mt-1" v-if="licenseStore.license?.days_remaining !== null && licenseStore.license?.days_remaining !== undefined">
          {{ licenseStore.license?.days_remaining }} días restantes
        </p>
      </div>
      <AppBadge :tone="statusTone(licenseStore.license?.status)">
        {{ licenseStore.license?.status ?? 'INACTIVE' }}
      </AppBadge>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <RouterLink :to="{ name: 'client-editor' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Editar mi perfil</p>
        <p class="text-xs text-gray-500 mt-1">Personaliza bloques, colores y plantilla.</p>
      </RouterLink>
      <RouterLink :to="{ name: 'client-qr' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Mi código QR</p>
        <p class="text-xs text-gray-500 mt-1">Personaliza y descarga tu QR.</p>
      </RouterLink>
      <RouterLink :to="{ name: 'client-stats' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Estadísticas</p>
        <p class="text-xs text-gray-500 mt-1">Visitas y clics en tu perfil.</p>
      </RouterLink>
    </div>
  </div>
</template>
