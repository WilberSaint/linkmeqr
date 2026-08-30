export type UserRole = 'ADMIN' | 'CLIENT'

export interface User {
  id: string
  email: string
  full_name: string
  phone: string | null
  role: UserRole
  is_active: boolean
  created_at: string
}

export type LicenseStatus = 'INACTIVE' | 'ACTIVE' | 'EXPIRED'

export interface License {
  status: LicenseStatus
  activated_at: string | null
  expires_at: string | null
  days_remaining: number | null
}

export interface ClientWithLicense extends User {
  license_status: LicenseStatus
  license_days_remaining: number | null
}

export type DurationType = '1_MONTH' | '3_MONTHS' | '6_MONTHS' | '1_YEAR' | 'CUSTOM'

export interface ActivationCode {
  id: string
  code: string
  duration_type: DurationType
  duration_days: number
  status: 'UNUSED' | 'USED' | 'REVOKED'
  batch_id: string | null
  assigned_user_id: string | null
  used_by_user_id: string | null
  created_at: string
  activated_at: string | null
  expires_at: string | null
  assigned_to_name: string | null
  assigned_to_email: string | null
  used_by_name: string | null
  used_by_email: string | null
}

export interface LicenseActivation {
  id: string
  activation_code_id: string
  code: string
  duration_days_added: number
  previous_expires_at: string | null
  new_expires_at: string
  activated_at: string
}

export type BlockType =
  | 'instagram'
  | 'facebook'
  | 'tiktok'
  | 'youtube'
  | 'whatsapp'
  | 'phone'
  | 'email'
  | 'location'
  | 'website'
  | 'menu'
  | 'catalog'
  | 'image'
  | 'video'
  | 'text'
  | 'link'
  | 'google_review'
  | 'gallery'
  | 'hours'
  | 'testimonials'
  | 'map'

export interface ProfileBlock {
  id: string
  profile_id: string
  block_type: BlockType
  title: string | null
  description: string | null
  url: string | null
  icon: string | null
  media_id: string | null
  media_url?: string | null
  style_overrides: Record<string, unknown> | null
  content: Record<string, unknown> | null
  is_visible: boolean
  sort_order: number
}

export interface ProfileTheme {
  background_type: 'color' | 'gradient' | 'pattern' | 'image'
  background_value: string
  background_media_id?: string | null
  background_url?: string | null
  primary_color: string
  secondary_color: string
  text_color: string
  button_text_color: string
  logo_background_color: string
  logo_text_color: string
  logo_display_mode: 'image' | 'initial'
  logo_shape: 'circle' | 'rounded' | 'square'
  font_family: string
  button_style: 'rounded' | 'square' | 'pill' | 'outline'
  button_shadow: boolean
  layout: 'list' | 'grid'
}

export interface Profile {
  id: string
  user_id: string
  slug: string
  business_name: string
  description: string | null
  logo_media_id: string | null
  cover_media_id: string | null
  logo_url?: string | null
  cover_url?: string | null
  template_id: string | null
  is_published: boolean
}

export interface PublicProfileResponse {
  inactive: boolean
  profile?: Profile
  theme?: ProfileTheme
  blocks?: ProfileBlock[]
}

export type QrPresetIcon = 'coffee' | 'heart' | 'matcha' | 'star' | 'gift'
export type QrFrameShape = 'heart' | 'coffee' | 'matcha' | 'circle' | 'star' | 'pizza' | 'flower' | 'cone' | 'custom_logo'

export interface QrCode {
  profile_id: string
  foreground_color: string
  background_color: string
  module_style: 'square' | 'dots' | 'rounded'
  eye_style: 'square' | 'circular' | 'rounded'
  logo_media_id: string | null
  logo_url: string | null
  logo_style: 'color' | 'monochrome' | 'dots'
  eye_color_from_logo: boolean
  preset_icon: QrPresetIcon | null
  frame_shape: QrFrameShape | null
  shape_fill: boolean
  error_correction: 'L' | 'M' | 'Q' | 'H'
  has_scannability_warning: boolean
}

export interface Template {
  id: string
  slug: string
  name: string
  description: string | null
  default_theme: ProfileTheme
  is_active: boolean
  sort_order: number
}

export interface MeResponse extends User {
  license: License
}

export interface ApiError {
  error: string
  message: string
  fields?: Record<string, string>
}
