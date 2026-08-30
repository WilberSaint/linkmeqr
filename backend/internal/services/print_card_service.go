package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

// randomScanCode is a short (10-char hex) public identifier for a card's
// /q/:code tracking link — deliberately much shorter than the card's own
// UUID, since every extra character here is extra density in the printed
// QR. 5 random bytes is plenty of collision headroom for a low-volume,
// per-business table like this one.
func randomScanCode() (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

var (
	ErrPrintCardNotOwner = errors.New("print card does not belong to this business")
	ErrNoMenuBlock       = errors.New("profile has no menu block configured")
	ErrBlockNotFound     = errors.New("block not found for this profile")
)

// linkableBlockTypes are the block types that represent a single meaningful
// "go here" destination — the ones worth offering as a QR target. Content
// blocks like gallery/hours/testimonials/map/text don't have one URL to
// point a QR at, so they're deliberately excluded.
var linkableBlockTypes = map[models.BlockType]bool{
	models.BlockInstagram:    true,
	models.BlockFacebook:     true,
	models.BlockTikTok:       true,
	models.BlockYouTube:      true,
	models.BlockWhatsapp:     true,
	models.BlockPhone:        true,
	models.BlockEmail:        true,
	models.BlockLocation:     true,
	models.BlockWebsite:      true,
	models.BlockMenu:         true,
	models.BlockCatalog:      true,
	models.BlockLink:         true,
	models.BlockGoogleReview: true,
}

type PrintCardService struct {
	repo      *repository.PrintCardRepository
	profiles  *ProfileService
	blocks    *BlockService
	qr        *QRManagementService
	loyalty   *LoyaltyService
	media     *repository.MediaRepository
	analytics *repository.AnalyticsRepository
}

func NewPrintCardService(repo *repository.PrintCardRepository, profiles *ProfileService, blocks *BlockService, qr *QRManagementService, loyalty *LoyaltyService, media *repository.MediaRepository, analytics *repository.AnalyticsRepository) *PrintCardService {
	return &PrintCardService{repo: repo, profiles: profiles, blocks: blocks, qr: qr, loyalty: loyalty, media: media, analytics: analytics}
}

// GetByScanCode loads a card by its short public scan code with no
// ownership check — used only by the public /q/:code redirect, which has no
// client/admin identity to check against and only ever uses the result to
// build a redirect.
func (s *PrintCardService) GetByScanCode(ctx context.Context, code string) (*models.PrintCard, error) {
	return s.repo.GetByScanCode(ctx, code)
}

// ScanCount is one card's total scan count — cheap enough to fetch
// individually wherever a single card is already being loaded.
func (s *PrintCardService) ScanCount(ctx context.Context, cardID string) (int, error) {
	return s.analytics.PrintCardScanCount(ctx, cardID)
}

// ScanCounts batches every one of a client's cards' scan counts into one
// query, keyed by card id — used by the list endpoint to avoid N+1 queries.
func (s *PrintCardService) ScanCounts(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := s.analytics.PrintCardScansByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.PrintCardID] = row.Count
	}
	return out, nil
}

func (s *PrintCardService) ListMine(ctx context.Context, userID string) ([]models.PrintCard, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Get loads a card and verifies it belongs to userID — every read/write
// entry point for an existing card goes through this.
func (s *PrintCardService) Get(ctx context.Context, userID, id string) (*models.PrintCard, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, ErrPrintCardNotOwner
	}
	return c, nil
}

type PrintCardInput struct {
	LayoutKey      models.PrintCardLayout
	Title          *string
	SizePreset     models.PrintCardSizePreset
	CustomWidthCm  *float64
	CustomHeightCm *float64
	QRTargetType   models.QRTargetType
	QRTargetValue  *string
	ColorOverrides *string
	Content        string
}

func (s *PrintCardService) Create(ctx context.Context, userID string, in PrintCardInput) (*models.PrintCard, error) {
	scanCode, err := randomScanCode()
	if err != nil {
		return nil, err
	}
	c := &models.PrintCard{
		ID:             uuid.NewString(),
		ScanCode:       scanCode,
		UserID:         userID,
		LayoutKey:      in.LayoutKey,
		Title:          in.Title,
		SizePreset:     in.SizePreset,
		CustomWidthCm:  in.CustomWidthCm,
		CustomHeightCm: in.CustomHeightCm,
		QRTargetType:   in.QRTargetType,
		QRTargetValue:  in.QRTargetValue,
		ColorOverrides: in.ColorOverrides,
		Content:        in.Content,
		Status:         models.SaleStatusDraft,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	// A new card gets its element tree immediately, seeded from the design
	// it was created with. Persisting it here (rather than lazily on first
	// open) means every card in the system has a real revision history from
	// version 1, and the editor never has to special-case "this card has no
	// layout yet".
	//
	// A failure to seed is deliberately not fatal: the card itself exists
	// and LayoutFor will seed the same tree on the fly next time it is
	// opened, so refusing the whole creation over it would be worse than
	// carrying on.
	if profile, err := s.profiles.GetByUserID(ctx, userID); err == nil {
		if layout, err := s.SeedLayoutFor(ctx, c, profile); err == nil {
			if encoded, err := json.Marshal(layout); err == nil {
				if version, err := s.repo.SaveLayout(ctx, c.ID, string(encoded), nil); err == nil {
					raw := string(encoded)
					c.Layout = &raw
					c.LayoutVersion = version
				}
			}
		}
	}
	return c, nil
}

func (s *PrintCardService) Update(ctx context.Context, userID, id string, in PrintCardInput) (*models.PrintCard, error) {
	c, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	c.LayoutKey = in.LayoutKey
	c.Title = in.Title
	c.SizePreset = in.SizePreset
	c.CustomWidthCm = in.CustomWidthCm
	c.CustomHeightCm = in.CustomHeightCm
	c.QRTargetType = in.QRTargetType
	c.QRTargetValue = in.QRTargetValue
	c.ColorOverrides = in.ColorOverrides
	c.Content = in.Content
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *PrintCardService) Delete(ctx context.Context, userID, id string) error {
	c, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, c.ID)
}

var validSaleStatuses = map[models.PrintCardSaleStatus]bool{
	models.SaleStatusDraft:     true,
	models.SaleStatusPrinted:   true,
	models.SaleStatusDelivered: true,
}

// UpdateStatus moves a card through LinkMeQR Studio's draft/printed/
// delivered pipeline — separate from Update so flipping a card's status
// never has to (and can't accidentally) touch its design.
func (s *PrintCardService) UpdateStatus(ctx context.Context, userID, id string, status models.PrintCardSaleStatus, saleNote *string) (*models.PrintCard, error) {
	c, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !validSaleStatuses[status] {
		return nil, errors.New("invalid sale status")
	}
	if err := s.repo.UpdateStatus(ctx, id, status, saleNote); err != nil {
		return nil, err
	}
	c.Status = status
	c.SaleNote = saleNote
	return c, nil
}

// QRTargetOption is one selectable QR destination offered to whoever is
// designing a card. Block-backed options carry the block's id (as
// TargetValue, ready to send straight back as qr_target_value) plus enough
// info for the frontend to label it using its own existing block-label
// logic instead of duplicating a label map here.
type QRTargetOption struct {
	TargetType  models.QRTargetType `json:"target_type"`
	TargetValue *string             `json:"target_value,omitempty"`
	BlockType   *models.BlockType   `json:"block_type,omitempty"`
	Title       *string             `json:"title,omitempty"`
}

// AvailableQRTargets enumerates what a card's QR can realistically point at
// for this specific profile — always the profile and loyalty card, one
// entry per existing block that has a real destination (so e.g. "Instagram"
// only ever appears when the business actually has that block), and a
// custom-URL escape hatch.
func (s *PrintCardService) AvailableQRTargets(ctx context.Context, profile *models.Profile) ([]QRTargetOption, error) {
	options := []QRTargetOption{
		{TargetType: models.QRTargetProfile},
		{TargetType: models.QRTargetLoyalty},
	}

	blocks, err := s.blocks.List(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		if !linkableBlockTypes[b.BlockType] {
			continue
		}
		hasURL := b.URL != nil && *b.URL != ""
		hasMedia := b.MediaID != nil
		if !hasURL && !hasMedia {
			continue
		}
		blockID, blockType := b.ID, b.BlockType
		options = append(options, QRTargetOption{TargetType: models.QRTargetBlock, TargetValue: &blockID, BlockType: &blockType, Title: b.Title})
	}

	options = append(options, QRTargetOption{TargetType: models.QRTargetCustomURL})
	return options, nil
}

// ResolveQRContent turns a card's qr_target_type/value into the actual URL
// its QR should encode.
func (s *PrintCardService) ResolveQRContent(ctx context.Context, targetType models.QRTargetType, targetValue *string, profile *models.Profile) (string, error) {
	switch targetType {
	case models.QRTargetProfile:
		return s.qr.ProfileURL(profile.Slug), nil
	case models.QRTargetLoyalty:
		program, err := s.loyalty.GetOrCreateProgram(ctx, profile.UserID)
		if err != nil {
			return "", err
		}
		return s.qr.LoyaltyURL(program.LoyaltyToken), nil
	case models.QRTargetMenu:
		blocks, err := s.blocks.List(ctx, profile.ID)
		if err != nil {
			return "", err
		}
		for _, b := range blocks {
			if b.BlockType != models.BlockMenu {
				continue
			}
			if url, err := s.blockDestination(ctx, &b); err == nil {
				return url, nil
			}
		}
		return "", ErrNoMenuBlock
	case models.QRTargetBlock:
		if targetValue == nil || *targetValue == "" {
			return "", errors.New("block target requires a block id")
		}
		block, err := s.blocks.Get(ctx, *targetValue)
		if err != nil {
			return "", err
		}
		if block.ProfileID != profile.ID {
			return "", ErrBlockNotFound
		}
		return s.blockDestination(ctx, block)
	case models.QRTargetCustomURL:
		if targetValue == nil || *targetValue == "" {
			return "", errors.New("custom_url target requires a value")
		}
		return *targetValue, nil
	default:
		return s.qr.ProfileURL(profile.Slug), nil
	}
}

// blockDestination resolves a single block's own URL (preferred) or, for
// blocks that only have an uploaded file (typically a menu PDF), its media
// file turned into an absolute URL — a relative "/media/..." path only
// resolves correctly inside the web app, not when scanned cold from a QR.
func (s *PrintCardService) blockDestination(ctx context.Context, block *models.ProfileBlock) (string, error) {
	if block.URL != nil && *block.URL != "" {
		return *block.URL, nil
	}
	if block.MediaID != nil {
		m, err := s.media.GetByID(ctx, *block.MediaID)
		if err == nil {
			return s.qr.AbsoluteURL(m.FilePath), nil
		}
	}
	return "", ErrBlockNotFound
}
