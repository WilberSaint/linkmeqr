<script setup lang="ts">
import { Globe, UtensilsCrossed, Gift, Link, Phone, Mail, MapPin, ShoppingBag, Star } from '@lucide/vue'
import type { QrTargetOption, QrTargetType } from '@/api/printCards'
import { blockLabel } from '@/composables/blockLabels'
import InstagramIcon from '@/components/icons/InstagramIcon.vue'
import FacebookIcon from '@/components/icons/FacebookIcon.vue'
import TiktokIcon from '@/components/icons/TiktokIcon.vue'
import YoutubeIcon from '@/components/icons/YoutubeIcon.vue'
import WhatsappIcon from '@/components/icons/WhatsappIcon.vue'

// Every option here is something that actually exists for this business —
// the list itself already excludes e.g. "Menú" when there's no menu block,
// so picking a target here can never point the QR at nothing.
const props = defineProps<{
  options: QrTargetOption[]
  targetType: QrTargetType
  targetValue: string
}>()
const emit = defineEmits<{ 'update:targetType': [v: QrTargetType]; 'update:targetValue': [v: string] }>()

const BLOCK_ICONS: Record<string, unknown> = {
  instagram: InstagramIcon,
  facebook: FacebookIcon,
  tiktok: TiktokIcon,
  youtube: YoutubeIcon,
  whatsapp: WhatsappIcon,
  phone: Phone,
  email: Mail,
  location: MapPin,
  website: Globe,
  menu: UtensilsCrossed,
  catalog: ShoppingBag,
  link: Link,
  google_review: Star,
}

function iconFor(opt: QrTargetOption) {
  if (opt.target_type === 'profile') return Globe
  if (opt.target_type === 'loyalty') return Gift
  if (opt.target_type === 'custom_url') return Link
  return BLOCK_ICONS[opt.block_type ?? ''] ?? Link
}

function labelFor(opt: QrTargetOption) {
  if (opt.target_type === 'profile') return 'Tu perfil completo'
  if (opt.target_type === 'loyalty') return 'Tu tarjeta de lealtad'
  if (opt.target_type === 'custom_url') return 'Link personalizado'
  return opt.title || blockLabel(opt.block_type ?? 'link')
}

function isSelected(opt: QrTargetOption) {
  if (opt.target_type !== props.targetType) return false
  if (opt.target_type === 'block') return opt.target_value === props.targetValue
  return true
}

function select(opt: QrTargetOption) {
  emit('update:targetType', opt.target_type)
  emit('update:targetValue', opt.target_type === 'block' ? opt.target_value ?? '' : '')
}
</script>

<template>
  <div class="space-y-1.5">
    <button
      v-for="(opt, i) in options"
      :key="i"
      type="button"
      class="w-full flex items-center gap-2.5 rounded-lg border-2 px-3 py-2 text-sm text-left transition"
      :class="isSelected(opt) ? 'border-indigo-500 bg-indigo-50 text-indigo-700' : 'border-gray-200 text-gray-600 hover:border-gray-300'"
      @click="select(opt)"
    >
      <component :is="iconFor(opt)" :size="16" class="shrink-0" />
      <span class="truncate">{{ labelFor(opt) }}</span>
    </button>
    <input
      v-if="targetType === 'custom_url'"
      :value="targetValue"
      placeholder="https://…"
      class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
      @input="emit('update:targetValue', ($event.target as HTMLInputElement).value)"
    />
  </div>
</template>
