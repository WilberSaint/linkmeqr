package models

import "time"

type Role string

const (
	RoleAdmin  Role = "ADMIN"
	RoleClient Role = "CLIENT"
)

type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role" json:"role"`
	FullName     string    `db:"full_name" json:"full_name"`
	Phone        *string   `db:"phone" json:"phone"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type RefreshToken struct {
	ID        string     `db:"id" json:"id"`
	UserID    string     `db:"user_id" json:"user_id"`
	TokenHash string     `db:"token_hash" json:"-"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at" json:"revoked_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

type LicenseStatus string

const (
	LicenseInactive LicenseStatus = "INACTIVE"
	LicenseActive   LicenseStatus = "ACTIVE"
	LicenseExpired  LicenseStatus = "EXPIRED"
)

type License struct {
	ID          string        `db:"id" json:"id"`
	UserID      string        `db:"user_id" json:"user_id"`
	Status      LicenseStatus `db:"status" json:"status"`
	ActivatedAt *time.Time    `db:"activated_at" json:"activated_at"`
	ExpiresAt   *time.Time    `db:"expires_at" json:"expires_at"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at" json:"updated_at"`
}

type DurationType string

const (
	Duration1Month  DurationType = "1_MONTH"
	Duration3Months DurationType = "3_MONTHS"
	Duration6Months DurationType = "6_MONTHS"
	Duration1Year   DurationType = "1_YEAR"
	DurationCustom  DurationType = "CUSTOM"
)

type CodeStatus string

const (
	CodeUnused  CodeStatus = "UNUSED"
	CodeUsed    CodeStatus = "USED"
	CodeRevoked CodeStatus = "REVOKED"
)

type ActivationCode struct {
	ID               string       `db:"id" json:"id"`
	Code             string       `db:"code" json:"code"`
	DurationType     DurationType `db:"duration_type" json:"duration_type"`
	DurationDays     int          `db:"duration_days" json:"duration_days"`
	Status           CodeStatus   `db:"status" json:"status"`
	BatchID          *string      `db:"batch_id" json:"batch_id"`
	AssignedUserID   *string      `db:"assigned_user_id" json:"assigned_user_id"`
	UsedByUserID     *string      `db:"used_by_user_id" json:"used_by_user_id"`
	CreatedByAdminID string       `db:"created_by_admin_id" json:"created_by_admin_id"`
	CreatedAt        time.Time    `db:"created_at" json:"created_at"`
	ActivatedAt      *time.Time   `db:"activated_at" json:"activated_at"`
	ExpiresAt        *time.Time   `db:"expires_at" json:"expires_at"`
	RevokedAt        *time.Time   `db:"revoked_at" json:"revoked_at"`
}

type LicenseActivation struct {
	ID                string     `db:"id" json:"id"`
	LicenseID         string     `db:"license_id" json:"license_id"`
	ActivationCodeID  string     `db:"activation_code_id" json:"activation_code_id"`
	UserID            string     `db:"user_id" json:"user_id"`
	DurationDaysAdded int        `db:"duration_days_added" json:"duration_days_added"`
	PreviousExpiresAt *time.Time `db:"previous_expires_at" json:"previous_expires_at"`
	NewExpiresAt      time.Time  `db:"new_expires_at" json:"new_expires_at"`
	ActivatedAt       time.Time  `db:"activated_at" json:"activated_at"`
}

type Template struct {
	ID           string    `db:"id" json:"id"`
	Slug         string    `db:"slug" json:"slug"`
	Name         string    `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description"`
	PreviewImage *string   `db:"preview_image" json:"preview_image"`
	DefaultTheme string    `db:"default_theme" json:"default_theme"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	SortOrder    int       `db:"sort_order" json:"sort_order"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Profile struct {
	ID           string    `db:"id" json:"id"`
	UserID       string    `db:"user_id" json:"user_id"`
	Slug         string    `db:"slug" json:"slug"`
	BusinessName string    `db:"business_name" json:"business_name"`
	Description  *string   `db:"description" json:"description"`
	LogoMediaID  *string   `db:"logo_media_id" json:"logo_media_id"`
	CoverMediaID *string   `db:"cover_media_id" json:"cover_media_id"`
	TemplateID   *string   `db:"template_id" json:"template_id"`
	IsPublished  bool      `db:"is_published" json:"is_published"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type ProfileTheme struct {
	ID                  string    `db:"id" json:"id"`
	ProfileID           string    `db:"profile_id" json:"profile_id"`
	BackgroundType      string    `db:"background_type" json:"background_type"`
	BackgroundValue     string    `db:"background_value" json:"background_value"`
	BackgroundMediaID   *string   `db:"background_media_id" json:"background_media_id"`
	PrimaryColor        string    `db:"primary_color" json:"primary_color"`
	SecondaryColor      string    `db:"secondary_color" json:"secondary_color"`
	TextColor           string    `db:"text_color" json:"text_color"`
	ButtonTextColor     string    `db:"button_text_color" json:"button_text_color"`
	LogoBackgroundColor string    `db:"logo_background_color" json:"logo_background_color"`
	LogoTextColor       string    `db:"logo_text_color" json:"logo_text_color"`
	LogoDisplayMode     string    `db:"logo_display_mode" json:"logo_display_mode"`
	LogoShape           string    `db:"logo_shape" json:"logo_shape"`
	FontFamily          string    `db:"font_family" json:"font_family"`
	ButtonStyle         string    `db:"button_style" json:"button_style"`
	ButtonShadow        bool      `db:"button_shadow" json:"button_shadow"`
	ExtraCSSVars        *string   `db:"extra_css_vars" json:"extra_css_vars"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

type BlockType string

const (
	BlockInstagram BlockType = "instagram"
	BlockFacebook  BlockType = "facebook"
	BlockTikTok    BlockType = "tiktok"
	BlockYouTube   BlockType = "youtube"
	BlockWhatsapp  BlockType = "whatsapp"
	BlockPhone     BlockType = "phone"
	BlockEmail     BlockType = "email"
	BlockLocation  BlockType = "location"
	BlockWebsite   BlockType = "website"
	BlockMenu      BlockType = "menu"
	BlockCatalog   BlockType = "catalog"
	BlockImage     BlockType = "image"
	BlockVideo     BlockType = "video"
	BlockText      BlockType = "text"
	BlockLink      BlockType = "link"
)

type ProfileBlock struct {
	ID             string    `db:"id" json:"id"`
	ProfileID      string    `db:"profile_id" json:"profile_id"`
	BlockType      BlockType `db:"block_type" json:"block_type"`
	Title          *string   `db:"title" json:"title"`
	Description    *string   `db:"description" json:"description"`
	URL            *string   `db:"url" json:"url"`
	Icon           *string   `db:"icon" json:"icon"`
	MediaID        *string   `db:"media_id" json:"media_id"`
	StyleOverrides *string   `db:"style_overrides" json:"style_overrides"`
	Content        *string   `db:"content" json:"content"`
	IsVisible      bool      `db:"is_visible" json:"is_visible"`
	SortOrder      int       `db:"sort_order" json:"sort_order"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type Media struct {
	ID          string    `db:"id" json:"id"`
	OwnerUserID string    `db:"owner_user_id" json:"owner_user_id"`
	FileName    string    `db:"file_name" json:"file_name"`
	FilePath    string    `db:"file_path" json:"file_path"`
	MimeType    string    `db:"mime_type" json:"mime_type"`
	SizeBytes   int64     `db:"size_bytes" json:"size_bytes"`
	Width       *int      `db:"width" json:"width"`
	Height      *int      `db:"height" json:"height"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type QRCode struct {
	ID                     string    `db:"id" json:"id"`
	ProfileID              string    `db:"profile_id" json:"profile_id"`
	ForegroundColor        string    `db:"foreground_color" json:"foreground_color"`
	BackgroundColor        string    `db:"background_color" json:"background_color"`
	ModuleStyle            string    `db:"module_style" json:"module_style"`
	EyeStyle               string    `db:"eye_style" json:"eye_style"`
	LogoMediaID            *string   `db:"logo_media_id" json:"logo_media_id"`
	ErrorCorrection        string    `db:"error_correction" json:"error_correction"`
	HasScannabilityWarning bool      `db:"has_scannability_warning" json:"has_scannability_warning"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
}

type EventType string

const (
	EventView       EventType = "VIEW"
	EventBlockClick EventType = "BLOCK_CLICK"
)

type AnalyticsEvent struct {
	ID          string    `db:"id" json:"id"`
	ProfileID   string    `db:"profile_id" json:"profile_id"`
	EventType   EventType `db:"event_type" json:"event_type"`
	BlockID     *string   `db:"block_id" json:"block_id"`
	DeviceType  *string   `db:"device_type" json:"device_type"`
	OSName      *string   `db:"os_name" json:"os_name"`
	BrowserName *string   `db:"browser_name" json:"browser_name"`
	Referrer    *string   `db:"referrer" json:"referrer"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type AuditLog struct {
	ID          string    `db:"id" json:"id"`
	ActorUserID *string   `db:"actor_user_id" json:"actor_user_id"`
	Action      string    `db:"action" json:"action"`
	EntityType  string    `db:"entity_type" json:"entity_type"`
	EntityID    *string   `db:"entity_id" json:"entity_id"`
	Metadata    *string   `db:"metadata" json:"metadata"`
	IPAddress   *string   `db:"ip_address" json:"ip_address"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
