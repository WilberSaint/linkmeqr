<script setup lang="ts">
import AppShell from '@/components/common/AppShell.vue'

const links = [
  { to: '/admin', label: 'Resumen' },
  { to: '/admin/clients', label: 'Clientes' },
  { to: '/admin/licenses', label: 'Licencias' },
  { to: '/admin/templates', label: 'Plantillas' },
  { to: '/admin/audit-logs', label: 'Auditoría' },
]
</script>

<template>
  <AppShell :links="links">
    <RouterView />
  </AppShell>
</template>
