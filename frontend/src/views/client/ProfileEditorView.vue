<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Check, Loader2 } from '@lucide/vue'
import { useEditorStore } from '@/stores/editor'
import * as templatesApi from '@/api/templates'
import * as profileApi from '@/api/profile'
import type { BlockType, ProfileBlock, ProfileTheme, Template } from '@/types'
import ProfilePreview from '@/components/public/ProfilePreview.vue'
import BlockList from '@/components/editor/BlockList.vue'
import ThemeEditor from '@/components/editor/ThemeEditor.vue'
import ImageCropModal from '@/components/editor/ImageCropModal.vue'

const editor = useEditorStore()
const router = useRouter()
const templates = ref<Template[]>([])
const tab = ref<'content' | 'design' | 'template'>('content')

const profileForm = ref({ business_name: '', description: '' })
const logoUploading = ref(false)
const backgroundUploading = ref(false)
const uploadingMenuFileFor = ref<string | null>(null)
const cropFile = ref<File | null>(null)
const cropOpen = ref(false)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const previewProfile = computed(() => ({
  business_name: profileForm.value.business_name,
  description: profileForm.value.description,
  logo_url: editor.profile?.logo_url,
}))

async function load() {
  await editor.loadAll()
  templates.value = await templatesApi.listTemplates()
  if (editor.profile) {
    profileForm.value = {
      business_name: editor.profile.business_name,
      description: editor.profile.description ?? '',
    }
  }
}

function currentProfilePayload(overrides: Partial<{ template_id: string | null; logo_media_id: string | null }> = {}) {
  if (!editor.profile) return null
  return {
    business_name: profileForm.value.business_name,
    description: profileForm.value.description,
    template_id: editor.profile.template_id,
    logo_media_id: editor.profile.logo_media_id,
    is_published: editor.profile.is_published,
    ...overrides,
  }
}

function scheduleProfileSave() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(async () => {
    const payload = currentProfilePayload()
    if (!payload) return
    await editor.saveProfile(payload)
  }, 600)
}

watch(() => profileForm.value.business_name, scheduleProfileSave)
watch(() => profileForm.value.description, scheduleProfileSave)

let themeDebounceTimer: ReturnType<typeof setTimeout> | null = null

// Color <input type="color"> fires an `input` event on every drag frame of
// the native picker, not just on release — without debouncing this sends a
// PATCH per frame. We update the local theme (and therefore the preview)
// immediately, but only persist to the server ~350ms after the user stops moving.
function onThemeUpdate(payload: Partial<ProfileTheme>) {
  if (!editor.theme) return
  const merged = { ...editor.theme, ...payload }
  editor.theme = merged

  if (themeDebounceTimer) clearTimeout(themeDebounceTimer)
  themeDebounceTimer = setTimeout(() => {
    editor.saveTheme(merged)
  }, 350)
}

async function onAddBlock(type: BlockType) {
  await editor.addBlock({ block_type: type, title: '', is_visible: true })
}

async function onUpdateBlock(id: string, payload: Partial<ProfileBlock>) {
  await editor.updateBlock(id, payload)
}

async function onRemoveBlock(id: string) {
  await editor.removeBlock(id)
}

async function onDuplicateBlock(id: string) {
  await editor.duplicateBlock(id)
}

async function onReorder(newList: ProfileBlock[]) {
  editor.reorderLocal(newList)
  await editor.persistOrder()
}

async function onSelectTemplate(templateId: string) {
  const payload = currentProfilePayload({ template_id: templateId })
  if (!payload) return
  await editor.saveProfile(payload)

  const tpl = templates.value.find((t) => t.id === templateId)
  if (tpl && editor.theme) {
    const merged: ProfileTheme = { ...editor.theme, ...tpl.default_theme }
    editor.theme = merged
    await editor.saveTheme(merged)
  }
}

function onLogoFileSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  ;(e.target as HTMLInputElement).value = ''
  if (!file) return
  cropFile.value = file
  cropOpen.value = true
}

async function onCropped(blob: Blob, shape: ProfileTheme['logo_shape']) {
  cropOpen.value = false
  logoUploading.value = true
  try {
    const file = new File([blob], 'logo.png', { type: 'image/png' })
    const media = await profileApi.uploadMedia(file)
    const payload = currentProfilePayload({ logo_media_id: media.id })
    if (payload) await editor.saveProfile(payload)
    // A freshly uploaded logo should show immediately without requiring the
    // user to also flip the display-mode toggle or shape by hand — both are
    // already decided in the crop modal. Saved immediately (not debounced)
    // since loadAll() right after would otherwise race a pending debounce
    // and overwrite this with the server's stale value.
    if (editor.theme) {
      const merged = { ...editor.theme, logo_display_mode: 'image' as const, logo_shape: shape }
      editor.theme = merged
      await editor.saveTheme(merged)
    }
    await editor.loadAll()
  } finally {
    logoUploading.value = false
    cropFile.value = null
  }
}

async function onUploadBackground(file: File) {
  if (!editor.theme) return
  backgroundUploading.value = true
  try {
    const media = await profileApi.uploadMedia(file)
    const merged = { ...editor.theme, background_type: 'image' as const, background_media_id: media.id }
    editor.theme = merged
    await editor.saveTheme(merged)
    await editor.loadAll()
  } finally {
    backgroundUploading.value = false
  }
}

async function onUploadMenuFile(blockId: string, file: File) {
  uploadingMenuFileFor.value = blockId
  try {
    const media = await profileApi.uploadMedia(file)
    await editor.updateBlock(blockId, { media_id: media.id })
  } finally {
    uploadingMenuFileFor.value = null
  }
}

onMounted(load)
</script>

<template>
  <div class="h-screen flex flex-col">
    <header class="shrink-0 h-12 px-4 flex items-center justify-between border-b border-gray-200 bg-white">
      <button
        type="button"
        class="flex items-center gap-1.5 text-sm text-gray-600 hover:text-gray-900"
        @click="router.push({ name: 'client-dashboard' })"
      >
        <ArrowLeft :size="16" />
        Volver
      </button>
      <div class="flex items-center gap-1.5 text-xs text-gray-400">
        <template v-if="editor.saveStatus === 'saving'">
          <Loader2 :size="13" class="animate-spin" />
          Guardando…
        </template>
        <template v-else-if="editor.saveStatus === 'saved'">
          <Check :size="13" class="text-green-600" />
          <span class="text-green-600">Guardado</span>
        </template>
        <template v-else>
          Los cambios se guardan automáticamente
        </template>
      </div>
    </header>

    <div class="flex-1 flex flex-col md:flex-row min-h-0">
    <div class="w-full md:w-96 shrink-0 border-b md:border-b-0 md:border-r border-gray-200 bg-white flex flex-col max-h-[45vh] md:max-h-none">
      <div class="px-4 pt-3 border-b border-gray-100 flex gap-4">
        <button
          v-for="t in [{ id: 'content', label: 'Contenido' }, { id: 'design', label: 'Diseño' }, { id: 'template', label: 'Plantilla' }]"
          :key="t.id"
          class="pb-2.5 text-sm font-medium border-b-2 transition-colors"
          :class="tab === t.id ? 'border-indigo-600 text-gray-900' : 'border-transparent text-gray-400 hover:text-gray-600'"
          @click="tab = t.id as typeof tab"
        >
          {{ t.label }}
        </button>
      </div>

      <div class="flex-1 overflow-y-auto p-4">
        <div v-if="tab === 'content'" class="space-y-5">
          <div class="space-y-3">
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1.5">Logo del negocio</label>
              <div class="flex items-center gap-3">
                <label class="relative group cursor-pointer shrink-0">
                  <img
                    v-if="editor.profile?.logo_url"
                    :src="editor.profile.logo_url"
                    alt="Logo"
                    class="w-16 h-16 rounded-full object-cover border border-gray-200"
                  />
                  <div
                    v-else
                    class="w-16 h-16 rounded-full flex items-center justify-center text-xl font-semibold text-white"
                    :style="{ backgroundColor: editor.theme?.logo_background_color }"
                  >
                    {{ (profileForm.business_name || '?').charAt(0).toUpperCase() }}
                  </div>
                  <div class="absolute inset-0 rounded-full bg-black/0 group-hover:bg-black/40 transition-colors flex items-center justify-center">
                    <span class="text-white text-[10px] font-medium opacity-0 group-hover:opacity-100 transition-opacity">
                      {{ logoUploading ? '…' : 'Cambiar' }}
                    </span>
                  </div>
                  <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" class="hidden" :disabled="logoUploading" @change="onLogoFileSelected" />
                </label>
                <div class="text-xs text-gray-400">PNG, JPG o WEBP<br />Recomendado: cuadrado</div>
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Nombre del negocio</label>
              <input v-model="profileForm.business_name" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Descripción</label>
              <textarea v-model="profileForm.description" rows="2" class="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
            </div>
          </div>

          <BlockList
            :blocks="editor.blocks"
            :uploading-menu-file-for="uploadingMenuFileFor"
            @add="onAddBlock"
            @update="onUpdateBlock"
            @remove="onRemoveBlock"
            @duplicate="onDuplicateBlock"
            @reorder="onReorder"
            @upload-menu-file="onUploadMenuFile"
          />
        </div>

        <ThemeEditor
          v-else-if="tab === 'design' && editor.theme"
          :theme="editor.theme"
          :has-logo-image="!!editor.profile?.logo_url"
          :background-uploading="backgroundUploading"
          @update="onThemeUpdate"
          @upload-background="onUploadBackground"
        />

        <div v-else-if="tab === 'template'" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <button
              v-for="t in templates"
              :key="t.id"
              class="relative border-2 rounded-xl overflow-hidden text-left transition hover:shadow-md"
              :class="editor.profile?.template_id === t.id ? 'border-indigo-500 ring-2 ring-indigo-100' : 'border-gray-200 hover:border-gray-300'"
              @click="onSelectTemplate(t.id)"
            >
              <div
                v-if="editor.profile?.template_id === t.id"
                class="absolute top-1.5 right-1.5 w-5 h-5 rounded-full bg-indigo-600 flex items-center justify-center z-10"
              >
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="3">
                  <path d="M20 6 9 17l-5-5" />
                </svg>
              </div>
              <div class="h-16 flex items-center justify-center gap-1.5 px-2" :style="{ background: t.default_theme.background_value }">
                <span
                  class="w-6 h-6 shrink-0"
                  :class="{
                    'rounded-full': t.default_theme.logo_shape === 'circle' || !t.default_theme.logo_shape,
                    'rounded-md': t.default_theme.logo_shape === 'rounded',
                    'rounded-none': t.default_theme.logo_shape === 'square',
                  }"
                  :style="{ backgroundColor: t.default_theme.logo_background_color }"
                />
                <span
                  class="text-[10px] px-2.5 py-1.5 font-medium"
                  :class="{
                    'rounded-full': t.default_theme.button_style === 'pill' || t.default_theme.button_style === 'rounded',
                    'rounded-none': t.default_theme.button_style === 'square',
                    'rounded border-2 bg-transparent!': t.default_theme.button_style === 'outline',
                  }"
                  :style="{
                    backgroundColor: t.default_theme.button_style === 'outline' ? 'transparent' : t.default_theme.secondary_color,
                    borderColor: t.default_theme.secondary_color,
                    color: t.default_theme.button_style === 'outline' ? t.default_theme.secondary_color : t.default_theme.button_text_color,
                  }"
                >
                  Botón
                </span>
              </div>
              <div class="p-2.5 bg-white">
                <p class="text-sm font-medium text-gray-900">{{ t.name }}</p>
                <p class="text-xs text-gray-500 mt-0.5 line-clamp-2">{{ t.description }}</p>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="flex-1 bg-gray-100 flex items-center justify-center p-4 md:p-8 min-h-0">
      <div class="w-full max-w-[360px] h-full max-h-[640px] md:h-[720px] rounded-[2rem] border-8 border-gray-900 overflow-hidden shadow-xl bg-white">
        <ProfilePreview :profile="previewProfile" :theme="editor.theme" :blocks="editor.blocks" />
      </div>
    </div>
    </div>

    <ImageCropModal
      :open="cropOpen"
      :file="cropFile"
      :shape="editor.theme?.logo_shape === 'square' ? 'square' : editor.theme?.logo_shape === 'rounded' ? 'rounded' : 'circle'"
      @close="cropOpen = false"
      @cropped="onCropped"
    />
  </div>
</template>
