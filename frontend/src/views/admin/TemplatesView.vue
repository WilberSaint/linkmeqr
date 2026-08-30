<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type { ProfileTheme, Template } from '@/types'
import * as templatesApi from '@/api/templates'
import { Palette, Plus } from '@lucide/vue'
import AppButton from '@/components/common/AppButton.vue'
import AppPageHeader from '@/components/common/AppPageHeader.vue'
import AppEmptyState from '@/components/common/AppEmptyState.vue'
import AppBadge from '@/components/common/AppBadge.vue'
import AppModal from '@/components/common/AppModal.vue'
import AppDropdownMenu from '@/components/common/AppDropdownMenu.vue'
import ThemeEditor from '@/components/editor/ThemeEditor.vue'

const DEFAULT_THEME: ProfileTheme = {
  background_type: 'color',
  background_value: '#f5f5f4',
  background_fit: 'cover',
  card_color: '#000000',
  card_opacity: 0.04,
  primary_color: '#111827',
  secondary_color: '#6366f1',
  text_color: '#111827',
  button_text_color: '#ffffff',
  logo_background_color: '#6366f1',
  logo_text_color: '#ffffff',
  logo_display_mode: 'initial',
  logo_shape: 'circle',
  font_family: 'Inter',
  button_style: 'rounded',
  button_shadow: false,
  layout: 'list',
}

const templates = ref<Template[]>([])
const loading = ref(false)

const editing = ref<Template | null>(null)
const showEditor = ref(false)
const slugTouched = ref(false)
const saving = ref(false)
const formError = ref('')

const form = ref({
  name: '',
  slug: '',
  description: '',
  sort_order: 0,
  theme: { ...DEFAULT_THEME },
})

async function load() {
  loading.value = true
  try {
    templates.value = await templatesApi.adminListTemplates()
  } finally {
    loading.value = false
  }
}

function slugify(name: string) {
  return name
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

watch(
  () => form.value.name,
  (name) => {
    if (!slugTouched.value) form.value.slug = slugify(name)
  },
)

function openCreate() {
  editing.value = null
  slugTouched.value = false
  formError.value = ''
  form.value = { name: '', slug: '', description: '', sort_order: templates.value.length, theme: { ...DEFAULT_THEME } }
  showEditor.value = true
}

function openEdit(t: Template) {
  editing.value = t
  slugTouched.value = true
  formError.value = ''
  form.value = {
    name: t.name,
    slug: t.slug,
    description: t.description ?? '',
    sort_order: t.sort_order,
    theme: { ...t.default_theme },
  }
  showEditor.value = true
}

function onThemeUpdate(payload: Partial<ProfileTheme>) {
  form.value.theme = { ...form.value.theme, ...payload }
}

async function onSave() {
  formError.value = ''
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      slug: form.value.slug,
      description: form.value.description || null,
      sort_order: form.value.sort_order,
      default_theme: form.value.theme,
    }
    if (editing.value) {
      await templatesApi.updateTemplate(editing.value.id, payload)
    } else {
      await templatesApi.createTemplate(payload)
    }
    showEditor.value = false
    await load()
  } catch {
    formError.value = 'No se pudo guardar la plantilla (¿slug ya en uso?).'
  } finally {
    saving.value = false
  }
}

async function onDelete(t: Template) {
  await templatesApi.deleteTemplate(t.id)
  await load()
}

async function onToggleActive(t: Template) {
  await templatesApi.setTemplateActive(t.id, !t.is_active)
  await load()
}

const hasLogoImage = computed(() => false)

onMounted(load)
</script>

<template>
  <div class="p-6 max-w-4xl">
    <AppPageHeader
      title="Plantillas de perfil"
      description="Combinaciones de colores, tipografía y botones que el cliente puede elegir como punto de partida para su perfil público. No afectan a las tarjetas impresas."
    >
      <template #actions>
        <AppButton @click="openCreate"><Plus :size="15" /> Nueva plantilla</AppButton>
      </template>
    </AppPageHeader>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div v-for="t in templates" :key="t.id" class="border border-gray-200 rounded-xl bg-white overflow-hidden">
        <div
          class="h-20 flex items-center justify-center"
          :style="{ background: t.default_theme.background_type === 'gradient' ? t.default_theme.background_value : t.default_theme.background_value }"
        >
          <span
            class="w-8 h-8 shrink-0"
            :class="{
              'rounded-full': t.default_theme.logo_shape === 'circle' || !t.default_theme.logo_shape,
              'rounded-md': t.default_theme.logo_shape === 'rounded',
              'rounded-none': t.default_theme.logo_shape === 'square',
            }"
            :style="{ backgroundColor: t.default_theme.logo_background_color }"
          />
        </div>
        <div class="p-4 flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <p class="text-sm font-semibold text-gray-900 truncate">{{ t.name }}</p>
              <AppBadge :tone="t.is_active ? 'green' : 'gray'">{{ t.is_active ? 'Activa' : 'Oculta' }}</AppBadge>
              <AppBadge tone="gray">{{ t.default_theme.layout === 'grid' ? 'Cuadrícula' : 'Lista' }}</AppBadge>
            </div>
            <p class="text-xs text-gray-500 mt-1 line-clamp-2">{{ t.description }}</p>
          </div>
          <AppDropdownMenu>
            <button type="button" class="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="openEdit(t)">
              Editar
            </button>
            <button type="button" class="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="onToggleActive(t)">
              {{ t.is_active ? 'Ocultar' : 'Mostrar' }}
            </button>
            <button type="button" class="w-full text-left px-3 py-2 text-sm text-red-600 hover:bg-red-50" @click="onDelete(t)">
              Eliminar
            </button>
          </AppDropdownMenu>
        </div>
      </div>
      <div v-if="!loading && templates.length === 0" class="col-span-2 rounded-xl border border-dashed border-gray-300 bg-white">
        <AppEmptyState
          :icon="Palette"
          title="Sin plantillas todavía"
          description="Crea una para que tus clientes tengan de dónde partir al armar su perfil, en vez de una página en blanco."
        >
          <template #action>
            <AppButton @click="openCreate"><Plus :size="15" /> Crear plantilla</AppButton>
          </template>
        </AppEmptyState>
      </div>
    </div>

    <AppModal :open="showEditor" :title="editing ? `Editar plantilla — ${editing.name}` : 'Nueva plantilla'" @close="showEditor = false">
      <form id="template-form" class="space-y-4" @submit.prevent="onSave">
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Nombre</label>
          <input v-model="form.name" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Slug</label>
          <input v-model="form.slug" required class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" @input="slugTouched = true" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 mb-1">Descripción</label>
          <textarea v-model="form.description" rows="2" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
        </div>
        <div class="border-t border-gray-100 pt-4">
          <ThemeEditor :theme="form.theme" :has-logo-image="hasLogoImage" @update="onThemeUpdate" />
        </div>

        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
      </form>

      <template #footer>
        <AppButton variant="secondary" type="button" @click="showEditor = false">Cancelar</AppButton>
        <AppButton type="submit" form="template-form" :disabled="saving">Guardar</AppButton>
      </template>
    </AppModal>
  </div>
</template>
