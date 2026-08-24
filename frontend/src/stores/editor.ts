import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Profile, ProfileBlock, ProfileTheme } from '@/types'
import * as profileApi from '@/api/profile'

export type SaveStatus = 'idle' | 'saving' | 'saved'

export const useEditorStore = defineStore('editor', () => {
  const profile = ref<Profile | null>(null)
  const theme = ref<ProfileTheme | null>(null)
  const blocks = ref<ProfileBlock[]>([])
  const loading = ref(false)
  const saveStatus = ref<SaveStatus>('idle')

  let savedResetTimer: ReturnType<typeof setTimeout> | null = null

  async function withSaveStatus<T>(fn: () => Promise<T>): Promise<T> {
    saveStatus.value = 'saving'
    try {
      const result = await fn()
      saveStatus.value = 'saved'
      if (savedResetTimer) clearTimeout(savedResetTimer)
      savedResetTimer = setTimeout(() => {
        saveStatus.value = 'idle'
      }, 2000)
      return result
    } catch (err) {
      saveStatus.value = 'idle'
      throw err
    }
  }

  async function loadAll() {
    loading.value = true
    try {
      const [p, t, b] = await Promise.all([
        profileApi.getMyProfile(),
        profileApi.getMyTheme(),
        profileApi.listMyBlocks(),
      ])
      profile.value = p
      theme.value = t
      blocks.value = b
    } finally {
      loading.value = false
    }
  }

  async function saveProfile(payload: {
    business_name: string
    description?: string | null
    template_id?: string | null
    logo_media_id?: string | null
    is_published: boolean
  }) {
    await withSaveStatus(async () => {
      profile.value = await profileApi.updateMyProfile(payload)
    })
  }

  async function saveTheme(payload: ProfileTheme) {
    await withSaveStatus(async () => {
      theme.value = await profileApi.updateMyTheme(payload)
    })
  }

  async function addBlock(payload: Partial<ProfileBlock>) {
    return withSaveStatus(async () => {
      const block = await profileApi.createBlock(payload)
      blocks.value.push(block)
      return block
    })
  }

  async function updateBlock(id: string, payload: Partial<ProfileBlock>) {
    return withSaveStatus(async () => {
      // The backend PATCH replaces the whole row from the JSON body (no
      // partial-merge semantics), so we must always send the full current
      // block state with just the requested fields changed — sending only the
      // delta would silently zero out unset fields like is_visible.
      const current = blocks.value.find((b) => b.id === id)
      const fullPayload = { ...current, ...payload }
      const updated = await profileApi.updateBlock(id, fullPayload)
      const idx = blocks.value.findIndex((b) => b.id === id)
      if (idx !== -1) blocks.value[idx] = updated
      return updated
    })
  }

  async function removeBlock(id: string) {
    await withSaveStatus(async () => {
      await profileApi.deleteBlock(id)
      blocks.value = blocks.value.filter((b) => b.id !== id)
    })
  }

  async function duplicateBlock(id: string) {
    return withSaveStatus(async () => {
      const block = await profileApi.duplicateBlock(id)
      blocks.value.push(block)
      return block
    })
  }

  async function persistOrder() {
    await withSaveStatus(async () => {
      const items = blocks.value.map((b, i) => ({ id: b.id, sort_order: i }))
      await profileApi.reorderBlocks(items)
    })
  }

  function reorderLocal(newOrder: ProfileBlock[]) {
    blocks.value = newOrder
  }

  return {
    profile,
    theme,
    blocks,
    loading,
    saveStatus,
    loadAll,
    saveProfile,
    saveTheme,
    addBlock,
    updateBlock,
    removeBlock,
    duplicateBlock,
    persistOrder,
    reorderLocal,
  }
})
