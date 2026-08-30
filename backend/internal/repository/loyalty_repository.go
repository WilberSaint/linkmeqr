package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type LoyaltyRepository struct {
	db *sqlx.DB
}

func NewLoyaltyRepository(db *sqlx.DB) *LoyaltyRepository {
	return &LoyaltyRepository{db: db}
}

// --- Programs ---

func (r *LoyaltyRepository) GetProgramByUserID(ctx context.Context, userID string) (*models.LoyaltyProgram, error) {
	var p models.LoyaltyProgram
	err := r.db.GetContext(ctx, &p, `SELECT * FROM loyalty_programs WHERE user_id = ?`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *LoyaltyRepository) GetProgramByToken(ctx context.Context, token string) (*models.LoyaltyProgram, error) {
	var p models.LoyaltyProgram
	err := r.db.GetContext(ctx, &p, `SELECT * FROM loyalty_programs WHERE loyalty_token = ?`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *LoyaltyRepository) CreateProgram(ctx context.Context, p *models.LoyaltyProgram) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO loyalty_programs (id, user_id, stamps_required, mid_reward_stamps, mid_reward_description, reward_description, loyalty_token, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.StampsRequired, p.MidRewardStamps, p.MidRewardDescription, p.RewardDescription, p.LoyaltyToken, p.IsActive,
	)
	return err
}

func (r *LoyaltyRepository) UpdateProgram(ctx context.Context, p *models.LoyaltyProgram) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE loyalty_programs SET
			stamps_required = ?, mid_reward_stamps = ?, mid_reward_description = ?, reward_description = ?, loyalty_token = ?, is_active = ?
		WHERE id = ?`,
		p.StampsRequired, p.MidRewardStamps, p.MidRewardDescription, p.RewardDescription, p.LoyaltyToken, p.IsActive, p.ID,
	)
	return err
}

// --- Customers ---

func (r *LoyaltyRepository) GetCustomerByIdentityToken(ctx context.Context, identityToken string) (*models.LoyaltyCustomer, error) {
	var c models.LoyaltyCustomer
	err := r.db.GetContext(ctx, &c, `SELECT * FROM loyalty_customers WHERE identity_token = ?`, identityToken)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LoyaltyRepository) GetCustomerByID(ctx context.Context, id string) (*models.LoyaltyCustomer, error) {
	var c models.LoyaltyCustomer
	err := r.db.GetContext(ctx, &c, `SELECT * FROM loyalty_customers WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *LoyaltyRepository) ListCustomersByUser(ctx context.Context, userID string) ([]models.LoyaltyCustomer, error) {
	customers := []models.LoyaltyCustomer{}
	err := r.db.SelectContext(ctx, &customers, `
		SELECT * FROM loyalty_customers WHERE user_id = ? ORDER BY updated_at DESC`, userID)
	return customers, err
}

func (r *LoyaltyRepository) CreateCustomer(ctx context.Context, c *models.LoyaltyCustomer) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO loyalty_customers (id, user_id, full_name, phone, identity_token, stamps_count)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.FullName, c.Phone, c.IdentityToken, c.StampsCount,
	)
	return err
}

func (r *LoyaltyRepository) IncrementStamps(ctx context.Context, customerID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE loyalty_customers SET stamps_count = stamps_count + 1 WHERE id = ?`, customerID)
	return err
}

func (r *LoyaltyRepository) ResetStamps(ctx context.Context, customerID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE loyalty_customers SET stamps_count = 0 WHERE id = ?`, customerID)
	return err
}

// --- Stamps (log) ---

func (r *LoyaltyRepository) CreateStamp(ctx context.Context, s *models.LoyaltyStamp) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO loyalty_stamps (id, loyalty_customer_id, source, created_by_admin_id)
		VALUES (?, ?, ?, ?)`,
		s.ID, s.LoyaltyCustomerID, s.Source, s.CreatedByAdminID,
	)
	return err
}

// LastStampAt returns the most recent stamp timestamp for a customer, used
// to enforce the anti-abuse cooldown — nil if they've never been stamped.
func (r *LoyaltyRepository) LastStampAt(ctx context.Context, customerID string) (*models.LoyaltyStamp, error) {
	var s models.LoyaltyStamp
	err := r.db.GetContext(ctx, &s, `
		SELECT * FROM loyalty_stamps WHERE loyalty_customer_id = ? ORDER BY created_at DESC LIMIT 1`, customerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
