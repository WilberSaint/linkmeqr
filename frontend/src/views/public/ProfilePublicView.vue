<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'
import * as publicApi from '@/api/public'
import ProfilePreview from '@/components/public/ProfilePreview.vue'
import ProfileInactiveView from './ProfileInactiveView.vue'

const route = useRoute()
const slug = route.params.slug as string

const loading = ref(true)
const inactive = ref(false)
const profile = ref<Profile | null>(null)
const theme = ref<ProfileTheme | null>(null)
const blocks = ref<ProfileBlock[]>([])

async function load() {
  loading.value = true
  try {
    const res = await publicApi.getPublicProfile(slug)
    if (res.inactive) {
      inactive.value = true
      return
    }
    profile.value = res.profile ?? null
    theme.value = res.theme ?? null
    blocks.value = res.blocks ?? []
  } catch {
    inactive.value = true
  } finally {
    loading.value = false
  }
}

function onBlockClick(blockId: string) {
  publicApi.trackEvent(slug, 'BLOCK_CLICK', blockId).catch(() => {})
}

onMounted(load)
</script>

<template>
  <div v-if="loading" class="min-h-screen flex items-center justify-center bg-gray-50">
    <p class="text-sm text-gray-400">Cargando…</p>
  </div>
  <ProfileInactiveView v-else-if="inactive" />
  <div v-else class="min-h-screen h-screen">
    <ProfilePreview :profile="profile" :theme="theme" :blocks="blocks" @block-click="onBlockClick" />
  </div>
</template>
