package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

type QRManagementService struct {
	repo          *repository.QRRepository
	media         *repository.MediaRepository
	mediaFiles    *MediaService
	publicBaseURL string
}

func NewQRManagementService(repo *repository.QRRepository, media *repository.MediaRepository, mediaFiles *MediaService, publicBaseURL string) *QRManagementService {
	return &QRManagementService{repo: repo, media: media, mediaFiles: mediaFiles, publicBaseURL: publicBaseURL}
}

// ProfileURL is the permanent URL a profile's QR always encodes, regardless
// of how the profile content is later edited.
func (s *QRManagementService) ProfileURL(slug string) string {
	return fmt.Sprintf("%s/p/%s", s.publicBaseURL, slug)
}

// AbsoluteURL turns a server-relative path (e.g. an uploaded media file's
// "/media/xxx.pdf") into a fully-qualified URL — needed anywhere that path
// gets encoded directly into a QR code, since a relative path only resolves
// correctly inside the web app itself, not when scanned cold from a phone.
func (s *QRManagementService) AbsoluteURL(path string) string {
	return s.publicBaseURL + path
}

// LoyaltyURL is the permanent URL a business's loyalty stamp card lives at —
// the same link a physical NFC tag would be programmed with, so a QR
// encoding this URL and an NFC tap both land the visitor on the same page.
func (s *QRManagementService) LoyaltyURL(token string) string {
	return fmt.Sprintf("%s/loyalty/%s", s.publicBaseURL, token)
}

// ScanURL is the short, trackable link a print card's exported QR encodes
// instead of its resolved destination directly — scanning it hits the
// public /q/:cardId[/:slot] redirect, which logs the scan and only then
// sends the visitor on to the real destination. slot is "" for every layout
// except multi_qr, which has two independent targets ("left"/"right") that
// need to be counted separately.
func (s *QRManagementService) ScanURL(cardID, slot string) string {
	if slot != "" {
		return fmt.Sprintf("%s/q/%s/%s", s.publicBaseURL, cardID, slot)
	}
	return fmt.Sprintf("%s/q/%s", s.publicBaseURL, cardID)
}

func (s *QRManagementService) GetOrCreate(ctx context.Context, profileID string) (*models.QRCode, error) {
	qr, err := s.repo.GetByProfileID(ctx, profileID)
	if err == nil {
		return qr, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	qr = &models.QRCode{
		ID:              uuid.NewString(),
		ProfileID:       profileID,
		ForegroundColor: "#000000",
		BackgroundColor: "#ffffff",
		ModuleStyle:     "square",
		EyeStyle:        "square",
		ErrorCorrection: "M",
		LogoStyle:       "color",
	}
	if err := s.repo.Create(ctx, qr); err != nil {
		return nil, err
	}
	return qr, nil
}

func (s *QRManagementService) Update(ctx context.Context, qr *models.QRCode) error {
	// A logo upload and a preset icon are mutually exclusive selections.
	// The frame shape is independent of both — it decorates the space
	// around the QR window, not the center badge.
	if qr.LogoMediaID != nil {
		qr.PresetIcon = nil
	} else if qr.PresetIcon != nil {
		qr.LogoMediaID = nil
		qr.LogoStyle = "color"
	} else {
		qr.LogoStyle = "color"
	}
	// Eye-color-from-logo only means something for a full-color logo — a
	// monochrome or dotted one is already drawn in the QR's own ink color,
	// so there'd be nothing distinct to sample.
	if qr.LogoMediaID == nil || qr.LogoStyle != "color" {
		qr.EyeColorFromLogo = false
	}
	// Shape-fill only means something once a frame shape is chosen.
	if qr.FrameShape == nil {
		qr.ShapeFill = false
	}

	frameShape := ""
	if qr.FrameShape != nil {
		frameShape = *qr.FrameShape
	}
	validation := Validate(QRCustomization{
		ForegroundColor: qr.ForegroundColor,
		BackgroundColor: qr.BackgroundColor,
		ModuleStyle:     qr.ModuleStyle,
		EyeStyle:        qr.EyeStyle,
		ErrorCorrection: QRErrorCorrection(qr.ErrorCorrection),
		HasLogo:         qr.LogoMediaID != nil || qr.PresetIcon != nil,
		FrameShape:      frameShape,
		ShapeFill:       qr.ShapeFill,
	})
	qr.ErrorCorrection = string(validation.EffectiveECLevel)
	qr.HasScannabilityWarning = len(validation.Warnings) > 0

	return s.repo.Update(ctx, qr)
}

func (s *QRManagementService) ToCustomization(ctx context.Context, qr *models.QRCode, slug string) QRCustomization {
	return s.ToCustomizationWithContent(ctx, qr, s.ProfileURL(slug))
}

// ToCustomizationWithContent applies a QR's saved styling to arbitrary
// content instead of the profile URL — used anywhere else in the app that
// wants "this business's QR look" pointed at a different destination (e.g.
// their loyalty card link, or a printable card's chosen target).
func (s *QRManagementService) ToCustomizationWithContent(ctx context.Context, qr *models.QRCode, content string) QRCustomization {
	c := QRCustomization{
		Content:         content,
		ForegroundColor: qr.ForegroundColor,
		BackgroundColor: qr.BackgroundColor,
		ModuleStyle:     qr.ModuleStyle,
		EyeStyle:        qr.EyeStyle,
		ErrorCorrection: QRErrorCorrection(qr.ErrorCorrection),
		HasLogo:         qr.LogoMediaID != nil || qr.PresetIcon != nil,
	}
	if qr.FrameShape != nil {
		c.FrameShape = *qr.FrameShape
		c.ShapeFill = qr.ShapeFill
	}

	if qr.PresetIcon != nil {
		c.PresetIcon = *qr.PresetIcon
		return c
	}

	if qr.LogoMediaID != nil {
		media, err := s.media.GetByID(ctx, *qr.LogoMediaID)
		if err != nil {
			return c
		}
		buf, err := s.mediaFiles.ReadFile(media)
		if err != nil {
			return c
		}
		img, _, err := image.Decode(bytes.NewReader(buf))
		if err != nil {
			return c
		}
		c.LogoImage = img
		c.LogoBytes = buf
		c.LogoMimeType = media.MimeType
		c.LogoStyle = qr.LogoStyle
		c.EyeColorFromLogo = qr.EyeColorFromLogo
	}

	return c
}
