package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

var (
	ErrLoyaltyProgramInactive = errors.New("loyalty program is not active")
	ErrLoyaltyNotOwner        = errors.New("customer does not belong to this business")
	ErrLoyaltyCardComplete    = errors.New("customer's card is already complete")
)

// stampCooldown is the minimum time between two stamps for the same
// customer — without this, someone could just refresh the NFC-tap page
// repeatedly to fill their card without actually visiting the business again.
const stampCooldown = 12 * time.Hour

type LoyaltyService struct {
	repo *repository.LoyaltyRepository
}

func NewLoyaltyService(repo *repository.LoyaltyRepository) *LoyaltyService {
	return &LoyaltyService{repo: repo}
}

func randomToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// GetOrCreateProgram returns the business's loyalty program, creating one
// with sensible defaults (and a fresh public token) the first time it's requested.
func (s *LoyaltyService) GetOrCreateProgram(ctx context.Context, userID string) (*models.LoyaltyProgram, error) {
	p, err := s.repo.GetProgramByUserID(ctx, userID)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	token, err := randomToken(12)
	if err != nil {
		return nil, err
	}
	p = &models.LoyaltyProgram{
		ID:             uuid.NewString(),
		UserID:         userID,
		StampsRequired: 10,
		LoyaltyToken:   token,
		IsActive:       true,
	}
	if err := s.repo.CreateProgram(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *LoyaltyService) UpdateProgram(ctx context.Context, p *models.LoyaltyProgram) error {
	return s.repo.UpdateProgram(ctx, p)
}

func (s *LoyaltyService) RegenerateToken(ctx context.Context, p *models.LoyaltyProgram) error {
	token, err := randomToken(12)
	if err != nil {
		return err
	}
	p.LoyaltyToken = token
	return s.repo.UpdateProgram(ctx, p)
}

func (s *LoyaltyService) ListCustomers(ctx context.Context, userID string) ([]models.LoyaltyCustomer, error) {
	return s.repo.ListCustomersByUser(ctx, userID)
}

func (s *LoyaltyService) ProgramByToken(ctx context.Context, token string) (*models.LoyaltyProgram, error) {
	return s.repo.GetProgramByToken(ctx, token)
}

func (s *LoyaltyService) CustomerByIdentityToken(ctx context.Context, token string) (*models.LoyaltyCustomer, error) {
	if token == "" {
		return nil, repository.ErrNotFound
	}
	return s.repo.GetCustomerByIdentityToken(ctx, token)
}

// RegisterCustomer creates a new end-customer for a business and records
// their first stamp in one step (the NFC tap that prompted registration
// counts as a visit).
func (s *LoyaltyService) RegisterCustomer(ctx context.Context, program *models.LoyaltyProgram, fullName string, phone *string) (*models.LoyaltyCustomer, error) {
	if !program.IsActive {
		return nil, ErrLoyaltyProgramInactive
	}

	identityToken, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	customer := &models.LoyaltyCustomer{
		ID:            uuid.NewString(),
		UserID:        program.UserID,
		FullName:      fullName,
		Phone:         phone,
		IdentityToken: identityToken,
		StampsCount:   1,
	}
	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		return nil, err
	}
	if err := s.repo.CreateStamp(ctx, &models.LoyaltyStamp{
		ID:                uuid.NewString(),
		LoyaltyCustomerID: customer.ID,
		Source:            models.StampSourceNFC,
	}); err != nil {
		return nil, err
	}
	return customer, nil
}

// StampIfEligible adds a stamp for an already-registered customer tapping
// the tag again, respecting the cooldown. Returns the (possibly unchanged)
// customer and whether a stamp was actually added.
func (s *LoyaltyService) StampIfEligible(ctx context.Context, customer *models.LoyaltyCustomer, program *models.LoyaltyProgram) (*models.LoyaltyCustomer, bool, error) {
	if !program.IsActive {
		return customer, false, ErrLoyaltyProgramInactive
	}
	// A completed card waits for the business to redeem it — otherwise a
	// customer could just keep tapping every cooldown window forever and
	// rack up stamps well past what the reward was meant to require.
	if customer.StampsCount >= program.StampsRequired {
		return customer, false, nil
	}

	last, err := s.repo.LastStampAt(ctx, customer.ID)
	if err != nil {
		return customer, false, err
	}
	if last != nil && time.Since(last.CreatedAt) < stampCooldown {
		return customer, false, nil
	}

	if err := s.repo.IncrementStamps(ctx, customer.ID); err != nil {
		return customer, false, err
	}
	if err := s.repo.CreateStamp(ctx, &models.LoyaltyStamp{
		ID:                uuid.NewString(),
		LoyaltyCustomerID: customer.ID,
		Source:            models.StampSourceNFC,
	}); err != nil {
		return customer, false, err
	}

	customer.StampsCount++
	return customer, true, nil
}

// ManualStamp lets the business add a stamp themselves (e.g. from the till),
// bypassing the cooldown since it's a deliberate staff action, not a tap.
// Refuses once the card has already reached stampsRequired — the next step
// for a complete card is Redeem, not more stamps, so a business clicking
// "Sellar" again by mistake shouldn't silently push the count past the
// card's own size.
func (s *LoyaltyService) ManualStamp(ctx context.Context, adminUserID string, customer *models.LoyaltyCustomer, stampsRequired int) error {
	if customer.UserID != adminUserID {
		return ErrLoyaltyNotOwner
	}
	if customer.StampsCount >= stampsRequired {
		return ErrLoyaltyCardComplete
	}
	if err := s.repo.IncrementStamps(ctx, customer.ID); err != nil {
		return err
	}
	if err := s.repo.CreateStamp(ctx, &models.LoyaltyStamp{
		ID:                uuid.NewString(),
		LoyaltyCustomerID: customer.ID,
		Source:            models.StampSourceManual,
		CreatedByAdminID:  &adminUserID,
	}); err != nil {
		return err
	}
	customer.StampsCount++
	return nil
}

func (s *LoyaltyService) Redeem(ctx context.Context, adminUserID string, customer *models.LoyaltyCustomer) error {
	if customer.UserID != adminUserID {
		return ErrLoyaltyNotOwner
	}
	return s.repo.ResetStamps(ctx, customer.ID)
}

func (s *LoyaltyService) CustomerByID(ctx context.Context, id string) (*models.LoyaltyCustomer, error) {
	return s.repo.GetCustomerByID(ctx, id)
}
