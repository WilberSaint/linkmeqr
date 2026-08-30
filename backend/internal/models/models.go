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
	Layout              string    `db:"layout" json:"layout"`
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

	BlockGoogleReview BlockType = "google_review"
	BlockGallery      BlockType = "gallery"
	BlockHours        BlockType = "hours"
	BlockTestimonials BlockType = "testimonials"
	BlockMap          BlockType = "map"
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
	LogoStyle              string    `db:"logo_style" json:"logo_style"`
	EyeColorFromLogo       bool      `db:"eye_color_from_logo" json:"eye_color_from_logo"`
	PresetIcon             *string   `db:"preset_icon" json:"preset_icon"`
	FrameShape             *string   `db:"frame_shape" json:"frame_shape"`
	ShapeFill              bool      `db:"shape_fill" json:"shape_fill"`
	ErrorCorrection        string    `db:"error_correction" json:"error_correction"`
	HasScannabilityWarning bool      `db:"has_scannability_warning" json:"has_scannability_warning"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
}

type EventType string

const (
	EventView       EventType = "VIEW"
	EventBlockClick EventType = "BLOCK_CLICK"
	EventQRScan     EventType = "QR_SCAN"
)

type AnalyticsEvent struct {
	ID          string    `db:"id" json:"id"`
	ProfileID   string    `db:"profile_id" json:"profile_id"`
	EventType   EventType `db:"event_type" json:"event_type"`
	BlockID     *string   `db:"block_id" json:"block_id"`
	PrintCardID *string   `db:"print_card_id" json:"print_card_id"`
	QRSlot      *string   `db:"qr_slot" json:"qr_slot"`
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

type LoyaltyProgram struct {
	ID                   string    `db:"id" json:"id"`
	UserID               string    `db:"user_id" json:"user_id"`
	StampsRequired       int       `db:"stamps_required" json:"stamps_required"`
	MidRewardStamps      *int      `db:"mid_reward_stamps" json:"mid_reward_stamps"`
	MidRewardDescription *string   `db:"mid_reward_description" json:"mid_reward_description"`
	RewardDescription    *string   `db:"reward_description" json:"reward_description"`
	LoyaltyToken         string    `db:"loyalty_token" json:"loyalty_token"`
	IsActive             bool      `db:"is_active" json:"is_active"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

type LoyaltyCustomer struct {
	ID            string    `db:"id" json:"id"`
	UserID        string    `db:"user_id" json:"user_id"`
	FullName      string    `db:"full_name" json:"full_name"`
	Phone         *string   `db:"phone" json:"phone"`
	IdentityToken string    `db:"identity_token" json:"-"`
	StampsCount   int       `db:"stamps_count" json:"stamps_count"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type StampSource string

const (
	StampSourceNFC    StampSource = "nfc"
	StampSourceManual StampSource = "manual"
)

type LoyaltyStamp struct {
	ID                string      `db:"id" json:"id"`
	LoyaltyCustomerID string      `db:"loyalty_customer_id" json:"loyalty_customer_id"`
	Source            StampSource `db:"source" json:"source"`
	CreatedByAdminID  *string     `db:"created_by_admin_id" json:"created_by_admin_id"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
}

// PrintCardLayout enumerates the built-in printable card layouts.
type PrintCardLayout string

const (
	PrintCardGoogleReview PrintCardLayout = "google_review"
	PrintCardSocialFollow PrintCardLayout = "social_follow"
	PrintCardMenuScan     PrintCardLayout = "menu_scan"
	PrintCardLoyaltyCard  PrintCardLayout = "loyalty_card"
	PrintCardMultiQR      PrintCardLayout = "multi_qr"
	PrintCardThankYou     PrintCardLayout = "thank_you"
)

// QRTargetType enumerates what URL a print card's QR (or, for multi_qr,
// each of its two QRs) encodes.
type QRTargetType string

const (
	QRTargetProfile   QRTargetType = "profile"
	QRTargetMenu      QRTargetType = "menu"
	QRTargetLoyalty   QRTargetType = "loyalty"
	QRTargetCustomURL QRTargetType = "custom_url"
	// QRTargetBlock points at one specific profile block (its QRTargetValue
	// is that block's id) — lets a card link straight to e.g. "the business's
	// Instagram" instead of only the generic profile/menu/loyalty shortcuts.
	QRTargetBlock QRTargetType = "block"
)

// PrintCardSizePreset enumerates the physical print sizes offered.
type PrintCardSizePreset string

const (
	SizeBusinessCard  PrintCardSizePreset = "business_card"
	SizeTableTent     PrintCardSizePreset = "table_tent"
	SizeStickerSquare PrintCardSizePreset = "sticker_square"
	SizeDoorHanger    PrintCardSizePreset = "door_hanger"
	// SizeCustom pairs with PrintCard.CustomWidthCm/CustomHeightCm instead
	// of a SizePresets lookup — the "editable size" escape hatch for
	// whatever a specific print shop or client actually needs.
	SizeCustom PrintCardSizePreset = "custom"
)

// PrintCardSaleStatus tracks a card through LinkMeQR Studio's own small
// sales pipeline — it's not just a design tool, it's how admin keeps track
// of what's been produced and handed over for each client.
type PrintCardSaleStatus string

const (
	SaleStatusDraft     PrintCardSaleStatus = "draft"
	SaleStatusPrinted   PrintCardSaleStatus = "printed"
	SaleStatusDelivered PrintCardSaleStatus = "delivered"
)

// PrintCard is a saved, printable marketing card design. ColorOverrides and
// Content are stored as raw JSON strings, same convention as
// Template.DefaultTheme / ProfileBlock.Content elsewhere in this file — the
// handler layer re-emits them as json.RawMessage so the frontend gets a
// real parsed object instead of a JSON-encoded string.
type PrintCard struct {
	ID         string              `db:"id" json:"id"`
	ScanCode   string              `db:"scan_code" json:"-"`
	UserID     string              `db:"user_id" json:"user_id"`
	LayoutKey  PrintCardLayout     `db:"layout_key" json:"layout_key"`
	Title      *string             `db:"title" json:"title"`
	SizePreset PrintCardSizePreset `db:"size_preset" json:"size_preset"`
	// CustomWidthCm/CustomHeightCm only apply when SizePreset == SizeCustom
	// — nil otherwise. See SizeCustom's own doc comment.
	CustomWidthCm  *float64            `db:"custom_width_cm" json:"custom_width_cm"`
	CustomHeightCm *float64            `db:"custom_height_cm" json:"custom_height_cm"`
	QRTargetType   QRTargetType        `db:"qr_target_type" json:"qr_target_type"`
	QRTargetValue  *string             `db:"qr_target_value" json:"qr_target_value"`
	ColorOverrides *string             `db:"color_overrides" json:"color_overrides"`
	Content        string              `db:"content" json:"content"`
	Status         PrintCardSaleStatus `db:"status" json:"status"`
	SaleNote       *string             `db:"sale_note" json:"sale_note"`

	// Layout is the card's element tree (a JSON-encoded CardLayout) and is
	// authoritative for both editing and export whenever it is set.
	// LayoutKey/Content/ColorOverrides above are the pre-tree model, kept
	// only so a card that has not been migrated yet can still be seeded;
	// nothing renders from them once Layout exists.
	Layout *string `db:"layout" json:"-"`
	// LayoutVersion is this card's own revision counter, incremented on
	// every layout save and matching a row in print_card_layout_versions.
	// Unrelated to models.CardLayoutVersion, which versions the schema.
	LayoutVersion int `db:"layout_version" json:"layout_version"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// PrintCardLayoutRevision is one saved revision of a card's element tree.
// Every layout save appends one, so a design can be rolled back after a bad
// edit — the printed artifact is the product being sold here, so losing a
// finished design to one stray drag is a real cost.
type PrintCardLayoutRevision struct {
	ID          string    `db:"id" json:"id"`
	PrintCardID string    `db:"print_card_id" json:"print_card_id"`
	Version     int       `db:"version" json:"version"`
	Layout      string    `db:"layout" json:"-"`
	CreatedBy   *string   `db:"created_by" json:"created_by"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
