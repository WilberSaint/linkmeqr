<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'
import * as publicApi from '@/api/public'
import ProfilePreview from '@/components/public/ProfilePreview.vue'
import ProfileInactiveView from './ProfileInactiveView.vue'
import ProfileNotFoundView from './ProfileNotFoundView.vue'

const route = useRoute()
const slug = route.params.slug as string

const publicUrl = `${window.location.origin}/p/${slug}`

const loading = ref(true)
const inactive = ref(false)
// The backend already tells these two apart — a slug that never existed
// (404) versus one that does but is unpublished or its license lapsed (200,
// inactive: true) — but the frontend used to collapse both into the same
// "Vuelve a intentarlo más tarde" message. That's actively wrong for a typo
// or an old QR pointing at a deleted profile: no amount of waiting fixes it.
const notFound = ref(false)
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
  } catch (err) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 404) {
      notFound.value = true
    } else {
      inactive.value = true
    }
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
  <ProfileNotFoundView v-else-if="notFound" />
  <div v-else class="min-h-screen h-screen">
    <ProfilePreview
      :profile="profile"
      :theme="theme"
      :blocks="blocks"
      :public-url="publicUrl"
      @block-click="onBlockClick"
    />
  </div>
</template>
