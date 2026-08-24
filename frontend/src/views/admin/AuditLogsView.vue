<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'

interface AuditLog {
  id: string
  actor_user_id: string | null
  action: string
  entity_type: string
  entity_id: string | null
  created_at: string
}

const logs = ref<AuditLog[]>([])
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    logs.value = await apiClient.get<AuditLog[]>('/admin/audit-logs').then((r) => r.data)
  } finally {
    loading.value = false
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('es-MX')
}

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-4xl">
    <h1 class="text-lg font-semibold text-gray-900 mb-6">Registro de auditoría</h1>

    <div class="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-gray-500 text-xs uppercase">
          <tr>
            <th class="text-left px-4 py-2.5 font-medium">Fecha</th>
            <th class="text-left px-4 py-2.5 font-medium">Acción</th>
            <th class="text-left px-4 py-2.5 font-medium">Entidad</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="log in logs" :key="log.id">
            <td class="px-4 py-2 text-gray-500">{{ formatDate(log.created_at) }}</td>
            <td class="px-4 py-2 text-gray-900">{{ log.action }}</td>
            <td class="px-4 py-2 text-gray-600">{{ log.entity_type }}</td>
          </tr>
          <tr v-if="!loading && logs.length === 0">
            <td colspan="3" class="px-4 py-8 text-center text-gray-400">Sin registros todavía.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
