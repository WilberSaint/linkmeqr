import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { License } from '@/types'
import * as authApi from '@/api/auth'

export const useLicenseStore = defineStore('license', () => {
  const license = ref<License | null>(null)
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      const me = await authApi.me()
      license.value = me.license
    } finally {
      loading.value = false
    }
  }

  return { license, loading, refresh }
})
