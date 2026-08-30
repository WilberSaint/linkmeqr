package services

import (
	"context"
	"encoding/json"
	"errors"

	"linkmeqr/backend/internal/models"
)

var (
	// ErrLayoutStale is returned when a save carries a base version that is
	// no longer the card's current one — two admins had the same card open
	// and the other one saved first.
	ErrLayoutStale = errors.New("layout has been modified by someone else")
	// ErrNoLayoutVersion is returned when a restore names a revision that
	// does not exist for this card.
	ErrNoLayoutVersion = errors.New("layout version not found")
)

// LayoutFor returns a card's element tree. A card that has been saved since
// the refactor has one stored; anything older is seeded on the fly from its
// pre-tree fields, so an un-backfilled card still renders and still opens in
// the editor. Seeding here is read-only — nothing is persisted until the
// designer actually saves, which keeps rendering a card free of write side
// effects.
func (s *PrintCardService) LayoutFor(ctx context.Context, card *models.PrintCard, profile *models.Profile) (*models.CardLayout, error) {
	if card.Layout != nil && *card.Layout != "" && *card.Layout != "null" {
		var layout models.CardLayout
		if err := json.Unmarshal([]byte(*card.Layout), &layout); err != nil {
			return nil, err
		}
		if err := layout.Validate(); err != nil {
			return nil, err
		}
		return &layout, nil
	}
	return s.SeedLayoutFor(ctx, card, profile)
}

// SeedLayoutFor generates the tree a card would start from, without saving
// it. Used both for the on-the-fly fallback above and by the backfill
// command, so the two can never drift apart.
func (s *PrintCardService) SeedLayoutFor(ctx context.Context, card *models.PrintCard, profile *models.Profile) (*models.CardLayout, error) {
	var theme *models.ProfileTheme
	hasLogo, logoShape := false, ""
	if profile != nil {
		theme, _ = s.profiles.GetTheme(ctx, profile.ID)
		hasLogo = profile.LogoMediaID != nil
		if theme != nil {
			logoShape = theme.LogoShape
		}
	}
	return SeedCardLayout(card, theme, hasLogo, logoShape), nil
}

// SaveLayout persists a new revision of a card's tree. baseVersion is the
// version the editor loaded; passing a stale one is refused rather than
// silently overwriting, because two admins designing the same physical card
// at once would otherwise lose one of their sessions' work entirely.
// Pass a nil baseVersion to skip the check (the backfill, which knows it is
// writing the very first revision).
func (s *PrintCardService) SaveLayout(ctx context.Context, userID, cardID string, layout *models.CardLayout, baseVersion *int, savedBy *string) (*models.PrintCard, error) {
	card, err := s.Get(ctx, userID, cardID)
	if err != nil {
		return nil, err
	}
	if baseVersion != nil && *baseVersion != card.LayoutVersion {
		return nil, ErrLayoutStale
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	layout.Version = models.CardLayoutVersion
	layout.NormalizeZ()

	encoded, err := json.Marshal(layout)
	if err != nil {
		return nil, err
	}
	next, err := s.repo.SaveLayout(ctx, cardID, string(encoded), savedBy)
	if err != nil {
		return nil, err
	}
	raw := string(encoded)
	card.Layout = &raw
	card.LayoutVersion = next
	return card, nil
}

// ListLayoutVersions returns a card's revision history, newest first.
func (s *PrintCardService) ListLayoutVersions(ctx context.Context, userID, cardID string) ([]models.PrintCardLayoutRevision, error) {
	if _, err := s.Get(ctx, userID, cardID); err != nil {
		return nil, err
	}
	return s.repo.ListLayoutVersions(ctx, cardID)
}

// RestoreLayoutVersion re-saves an earlier revision as a NEW revision rather
// than rewinding the counter — the history stays append-only, so restoring
// is itself undoable.
func (s *PrintCardService) RestoreLayoutVersion(ctx context.Context, userID, cardID string, version int, restoredBy *string) (*models.CardLayout, *models.PrintCard, error) {
	if _, err := s.Get(ctx, userID, cardID); err != nil {
		return nil, nil, err
	}
	rev, err := s.repo.GetLayoutVersion(ctx, cardID, version)
	if err != nil {
		return nil, nil, ErrNoLayoutVersion
	}
	var layout models.CardLayout
	if err := json.Unmarshal([]byte(rev.Layout), &layout); err != nil {
		return nil, nil, err
	}
	card, err := s.SaveLayout(ctx, userID, cardID, &layout, nil, restoredBy)
	if err != nil {
		return nil, nil, err
	}
	return &layout, card, nil
}

// LayoutAssetSource is everything the renderer's assets are resolved
// against. Bundled into one struct because resolving a tree needs several
// unrelated lookups (the profile for its logo, the QR style for module
// shapes, the media store for uploads) and threading them individually
// through every call site was the main thing making the old render path
// hard to follow.
type LayoutAssetSource struct {
	Profile *models.Profile
	QRStyle *models.QRCode
	// Tracked makes every QR encode the card's own /q/:code[/:slot] short
	// link instead of its destination directly, so scanning a printed card
	// is counted before the visitor is forwarded. Only ever true for a real
	// export of a saved card — a live preview has nothing to track against.
	Tracked  bool
	ScanCode string
}

// ResolveLayoutAssets renders each QR element's own code and loads every
// image the tree references, producing the asset bundle RenderLayoutSVG
// needs. Each QR resolves independently, so a tree can hold any number of
// them pointing anywhere — the old model could only ever express one, or
// exactly two in the special multi_qr case.
func (s *PrintCardService) ResolveLayoutAssets(ctx context.Context, layout *models.CardLayout, src LayoutAssetSource, mediaSvc *MediaService) (*LayoutAssets, error) {
	assets := &LayoutAssets{QRSVGs: map[string]string{}, Images: map[string]ImageAsset{}}

	if src.Profile != nil && src.Profile.LogoMediaID != nil && mediaSvc != nil {
		if m, err := s.media.GetByID(ctx, *src.Profile.LogoMediaID); err == nil {
			if buf, err := mediaSvc.ReadFile(m); err == nil {
				assets.Logo = &ImageAsset{Bytes: buf, MimeType: m.MimeType}
			}
		}
	}

	for _, el := range layout.Elements {
		if el.Hidden {
			continue
		}
		switch el.Type {
		case models.ElementQR:
			var p models.QRProps
			if err := el.DecodeProps(&p); err != nil {
				continue
			}
			content := ""
			if src.Tracked && src.ScanCode != "" {
				content = s.qr.ScanURL(src.ScanCode, el.ScanSlot())
			} else {
				resolved, err := s.ResolveQRContent(ctx, p.TargetType, p.TargetValue, src.Profile)
				if err != nil {
					// A single unresolvable destination must not fail the
					// whole render: the element falls back to its
					// placeholder box so the designer can see which QR is
					// misconfigured instead of getting an error page for
					// the entire card.
					continue
				}
				content = resolved
			}
			svg, err := RenderSVG(s.qr.ToCustomizationWithContent(ctx, src.QRStyle, content))
			if err != nil {
				continue
			}
			assets.QRSVGs[el.ID] = svg

		case models.ElementImage:
			var p models.ImageProps
			if err := el.DecodeProps(&p); err != nil || p.Source == "logo" || p.MediaID == "" {
				continue
			}
			m, err := s.media.GetByID(ctx, p.MediaID)
			if err != nil || mediaSvc == nil {
				continue
			}
			// Only ever serve media the business owns — an element could
			// otherwise name any media id in the system and have the
			// exporter embed it.
			if src.Profile != nil && m.OwnerUserID != src.Profile.UserID {
				continue
			}
			buf, err := mediaSvc.ReadFile(m)
			if err != nil {
				continue
			}
			assets.Images[el.ID] = ImageAsset{Bytes: buf, MimeType: m.MimeType}
		}
	}

	return assets, nil
}
