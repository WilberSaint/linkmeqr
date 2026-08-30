<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import { ArrowLeftCircle } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function onReturnToAdmin() {
  const clientId = auth.user?.id
  auth.stopImpersonation()
  router.push(clientId ? { name: 'admin-client-detail', params: { id: clientId } } : { name: 'admin-dashboard' })
}
</script>

<template>
  <div v-if="auth.isImpersonating" class="sticky top-0 z-50 bg-amber-500 text-white text-sm px-4 py-2 flex items-center justify-between">
    <span>Viendo como <strong>{{ auth.user?.full_name }}</strong> — los cambios que hagas se guardan en su cuenta.</span>
    <button type="button" class="inline-flex items-center gap-1.5 font-medium hover:underline shrink-0 ml-3" @click="onReturnToAdmin">
      <ArrowLeftCircle :size="16" />
      Volver a admin
    </button>
  </div>
  <RouterView />
</template>
