package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

type QRManagementService struct {
	repo          *repository.QRRepository
	publicBaseURL string
}

func NewQRManagementService(repo *repository.QRRepository, publicBaseURL string) *QRManagementService {
	return &QRManagementService{repo: repo, publicBaseURL: publicBaseURL}
}

// ProfileURL is the permanent URL a profile's QR always encodes, regardless
// of how the profile content is later edited.
func (s *QRManagementService) ProfileURL(slug string) string {
	return fmt.Sprintf("%s/p/%s", s.publicBaseURL, slug)
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
	}
	if err := s.repo.Create(ctx, qr); err != nil {
		return nil, err
	}
	return qr, nil
}

func (s *QRManagementService) Update(ctx context.Context, qr *models.QRCode) error {
	validation := Validate(QRCustomization{
		ForegroundColor: qr.ForegroundColor,
		BackgroundColor: qr.BackgroundColor,
		ModuleStyle:     qr.ModuleStyle,
		EyeStyle:        qr.EyeStyle,
		ErrorCorrection: QRErrorCorrection(qr.ErrorCorrection),
		HasLogo:         qr.LogoMediaID != nil,
	})
	qr.ErrorCorrection = string(validation.EffectiveECLevel)
	qr.HasScannabilityWarning = len(validation.Warnings) > 0

	return s.repo.Update(ctx, qr)
}

func (s *QRManagementService) ToCustomization(qr *models.QRCode, slug string) QRCustomization {
	return QRCustomization{
		Content:         s.ProfileURL(slug),
		ForegroundColor: qr.ForegroundColor,
		BackgroundColor: qr.BackgroundColor,
		ModuleStyle:     qr.ModuleStyle,
		EyeStyle:        qr.EyeStyle,
		ErrorCorrection: QRErrorCorrection(qr.ErrorCorrection),
		HasLogo:         qr.LogoMediaID != nil,
	}
}
