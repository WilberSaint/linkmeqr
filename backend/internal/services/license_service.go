package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

var (
	ErrCodeNotFound  = errors.New("activation code not found")
	ErrCodeNotUsable = errors.New("activation code already used or revoked")
	// ErrCodeNotYours is a code reserved for a different client. Deliberately
	// distinct from "not found": telling the wrong person a code exists but is
	// not theirs is far more useful than a blanket denial, and reveals nothing
	// they could not already infer by trying it.
	ErrCodeNotYours = errors.New("activation code belongs to another client")
)

type LicenseService struct {
	db          *sqlx.DB
	licenses    *repository.LicenseRepository
	codes       *repository.ActivationCodeRepository
	activations *repository.LicenseActivationRepository
	audit       *AuditService
}

func NewLicenseService(
	db *sqlx.DB,
	licenses *repository.LicenseRepository,
	codes *repository.ActivationCodeRepository,
	activations *repository.LicenseActivationRepository,
	audit *AuditService,
) *LicenseService {
	return &LicenseService{db: db, licenses: licenses, codes: codes, activations: activations, audit: audit}
}

func DurationDays(durationType models.DurationType, customDays int) (int, error) {
	switch durationType {
	case models.Duration1Month:
		return 30, nil
	case models.Duration3Months:
		return 90, nil
	case models.Duration6Months:
		return 180, nil
	case models.Duration1Year:
		return 365, nil
	case models.DurationCustom:
		if customDays <= 0 {
			return 0, fmt.Errorf("custom duration requires a positive number of days")
		}
		return customDays, nil
	default:
		return 0, fmt.Errorf("unknown duration type: %s", durationType)
	}
}

// AssignCode reserves an unused code for one client, or releases it when
// userID is nil.
//
// Codes are bearer tokens by default: whoever types one first claims it. That
// is right for a printed batch handed out at an event, and wrong for a code
// generated for one paying client — assigning it means only that client can
// redeem it, so a leaked code cannot be spent by somebody else.
func (s *LicenseService) AssignCode(ctx context.Context, codeID string, userID *string) error {
	return s.codes.Assign(ctx, codeID, userID)
}

// GenerateCode creates a single activation code.
func (s *LicenseService) GenerateCode(ctx context.Context, adminID string, durationType models.DurationType, customDays int) (*models.ActivationCode, error) {
	days, err := DurationDays(durationType, customDays)
	if err != nil {
		return nil, err
	}

	code, err := randomCode()
	if err != nil {
		return nil, err
	}

	ac := &models.ActivationCode{
		ID:               uuid.NewString(),
		Code:             code,
		DurationType:     durationType,
		DurationDays:     days,
		Status:           models.CodeUnused,
		CreatedByAdminID: adminID,
	}
	if err := s.codes.Create(ctx, ac); err != nil {
		return nil, err
	}
	return ac, nil
}

// GenerateBatch creates `quantity` activation codes sharing a batch ID.
func (s *LicenseService) GenerateBatch(ctx context.Context, adminID string, durationType models.DurationType, customDays, quantity int) ([]models.ActivationCode, error) {
	if quantity <= 0 || quantity > 5000 {
		return nil, fmt.Errorf("quantity must be between 1 and 5000")
	}

	days, err := DurationDays(durationType, customDays)
	if err != nil {
		return nil, err
	}

	batchID := uuid.NewString()
	codes := make([]models.ActivationCode, 0, quantity)
	seen := map[string]bool{}

	for len(codes) < quantity {
		code, err := randomCode()
		if err != nil {
			return nil, err
		}
		if seen[code] {
			continue
		}
		seen[code] = true

		codes = append(codes, models.ActivationCode{
			ID:               uuid.NewString(),
			Code:             code,
			DurationType:     durationType,
			DurationDays:     days,
			Status:           models.CodeUnused,
			BatchID:          &batchID,
			CreatedByAdminID: adminID,
		})
	}

	if err := s.codes.CreateBatch(ctx, codes); err != nil {
		return nil, err
	}
	return codes, nil
}

// ActivateCode is the core licensing algorithm:
//   - if the client has no license yet, or their current license is expired,
//     the new duration counts from now;
//   - if the client's license is still active, the new duration is appended
//     to the existing expiration date instead of replacing it.
//
// Runs inside a single transaction with SELECT ... FOR UPDATE on both the
// code and the license row to make concurrent activation attempts safe.
func (s *LicenseService) ActivateCode(ctx context.Context, userID, code string) (*models.License, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ac, err := s.codes.GetByCodeForUpdate(ctx, tx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCodeNotFound
		}
		return nil, err
	}
	if ac.Status != models.CodeUnused {
		return nil, ErrCodeNotUsable
	}
	// A code reserved for one client is not a bearer token any more: handing
	// it to somebody else must not silently transfer the licence they paid
	// for.
	if ac.AssignedUserID != nil && *ac.AssignedUserID != userID {
		return nil, ErrCodeNotYours
	}

	now := time.Now().UTC()

	license, err := s.licenses.GetByUserIDForUpdate(ctx, tx, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	var previousExpiresAt *time.Time
	var newExpiresAt time.Time

	if license == nil {
		newExpiresAt = now.AddDate(0, 0, ac.DurationDays)
		license = &models.License{
			ID:          uuid.NewString(),
			UserID:      userID,
			Status:      models.LicenseActive,
			ActivatedAt: &now,
			ExpiresAt:   &newExpiresAt,
		}
		if err := s.licenses.CreateTx(ctx, tx, license); err != nil {
			return nil, err
		}
	} else {
		previousExpiresAt = license.ExpiresAt
		stillActive := license.Status == models.LicenseActive && license.ExpiresAt != nil && license.ExpiresAt.After(now)

		if stillActive {
			newExpiresAt = license.ExpiresAt.AddDate(0, 0, ac.DurationDays)
		} else {
			newExpiresAt = now.AddDate(0, 0, ac.DurationDays)
		}

		license.Status = models.LicenseActive
		license.ExpiresAt = &newExpiresAt
		if license.ActivatedAt == nil {
			license.ActivatedAt = &now
		}
		if err := s.licenses.UpdateTx(ctx, tx, license); err != nil {
			return nil, err
		}
	}

	ac.Status = models.CodeUsed
	ac.UsedByUserID = &userID
	ac.ActivatedAt = &now
	ac.ExpiresAt = &newExpiresAt
	if err := s.codes.MarkUsedTx(ctx, tx, ac); err != nil {
		return nil, err
	}

	activation := &models.LicenseActivation{
		ID:                uuid.NewString(),
		LicenseID:         license.ID,
		ActivationCodeID:  ac.ID,
		UserID:            userID,
		DurationDaysAdded: ac.DurationDays,
		PreviousExpiresAt: previousExpiresAt,
		NewExpiresAt:      newExpiresAt,
		ActivatedAt:       now,
	}
	if err := s.activations.CreateTx(ctx, tx, activation); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return license, nil
}

func (s *LicenseService) ListCodes(ctx context.Context, filter repository.ListCodesFilter) ([]models.ActivationCode, error) {
	return s.codes.List(ctx, filter)
}

// ActivateForClient lets an admin generate a code and apply it to a client's
// license in one step, instead of the admin generating a code and the client
// separately entering it — used by the "Activar licencia" action in the
// admin clients table.
func (s *LicenseService) ActivateForClient(ctx context.Context, adminID, clientUserID string, durationType models.DurationType, customDays int) (*models.License, error) {
	ac, err := s.GenerateCode(ctx, adminID, durationType, customDays)
	if err != nil {
		return nil, err
	}
	return s.ActivateCode(ctx, clientUserID, ac.Code)
}

func (s *LicenseService) RevokeCode(ctx context.Context, id string) error {
	return s.codes.Revoke(ctx, id)
}

// EffectiveStatus resolves the lazily-computed status: a license whose
// expires_at has passed is treated as EXPIRED even if the stored status
// column hasn't been swept yet.
func EffectiveStatus(l *models.License) models.LicenseStatus {
	if l == nil {
		return models.LicenseInactive
	}
	if l.Status == models.LicenseActive && l.ExpiresAt != nil && l.ExpiresAt.Before(time.Now().UTC()) {
		return models.LicenseExpired
	}
	return l.Status
}

func DaysRemaining(l *models.License) *int {
	if l == nil || l.ExpiresAt == nil {
		return nil
	}
	days := int(time.Until(*l.ExpiresAt).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return &days
}

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no O/0/I/1 to avoid ambiguity on printed cards

func randomCode() (string, error) {
	const groups = 4
	const groupLen = 4
	out := make([]byte, 0, groups*groupLen+groups-1)

	for g := 0; g < groups; g++ {
		if g > 0 {
			out = append(out, '-')
		}
		for i := 0; i < groupLen; i++ {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
			if err != nil {
				return "", err
			}
			out = append(out, codeAlphabet[n.Int64()])
		}
	}
	return string(out), nil
}
