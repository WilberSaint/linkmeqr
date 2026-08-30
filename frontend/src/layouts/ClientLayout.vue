<script setup lang="ts">
import AppShell from '@/components/common/AppShell.vue'

const links = [
  { to: '/client', label: 'Resumen' },
  { to: '/client/editor', label: 'Editor de perfil' },
  { to: '/client/license', label: 'Licencia' },
  { to: '/client/loyalty', label: 'Lealtad' },
]
</script>

<template>
  <AppShell :links="links">
    <RouterView />
  </AppShell>
</template>
