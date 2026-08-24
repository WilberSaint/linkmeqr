<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="p-6 max-w-3xl space-y-6">
    <div>
      <h1 class="text-lg font-semibold text-gray-900">Panel administrativo</h1>
      <p class="text-sm text-gray-500 mt-1">{{ auth.user?.full_name }} — LinkMeQR Admin</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <RouterLink :to="{ name: 'admin-clients' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Clientes</p>
        <p class="text-xs text-gray-500 mt-1">Crear, activar y administrar clientes.</p>
      </RouterLink>
      <RouterLink :to="{ name: 'admin-licenses' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Licencias</p>
        <p class="text-xs text-gray-500 mt-1">Generar códigos individuales o por lote.</p>
      </RouterLink>
      <RouterLink :to="{ name: 'admin-templates' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Plantillas</p>
        <p class="text-xs text-gray-500 mt-1">Ver plantillas visuales disponibles.</p>
      </RouterLink>
      <RouterLink :to="{ name: 'admin-audit' }" class="bg-white border border-gray-200 rounded-xl p-5 hover:border-indigo-300">
        <p class="text-sm font-medium text-gray-900">Auditoría</p>
        <p class="text-xs text-gray-500 mt-1">Historial de acciones administrativas.</p>
      </RouterLink>
    </div>
  </div>
</template>
