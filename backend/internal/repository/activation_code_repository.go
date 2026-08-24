package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type ActivationCodeRepository struct {
	db *sqlx.DB
}

func NewActivationCodeRepository(db *sqlx.DB) *ActivationCodeRepository {
	return &ActivationCodeRepository{db: db}
}

func (r *ActivationCodeRepository) Create(ctx context.Context, c *models.ActivationCode) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO activation_codes
			(id, code, duration_type, duration_days, status, batch_id, assigned_user_id, created_by_admin_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Code, c.DurationType, c.DurationDays, c.Status, c.BatchID, c.AssignedUserID, c.CreatedByAdminID,
	)
	return err
}

func (r *ActivationCodeRepository) CreateBatch(ctx context.Context, codes []models.ActivationCode) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO activation_codes
			(id, code, duration_type, duration_days, status, batch_id, created_by_admin_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range codes {
		if _, err := stmt.ExecContext(ctx, c.ID, c.Code, c.DurationType, c.DurationDays, c.Status, c.BatchID, c.CreatedByAdminID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *ActivationCodeRepository) GetByCodeForUpdate(ctx context.Context, tx *sqlx.Tx, code string) (*models.ActivationCode, error) {
	var c models.ActivationCode
	err := tx.GetContext(ctx, &c, `SELECT * FROM activation_codes WHERE code = ? FOR UPDATE`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ActivationCodeRepository) MarkUsedTx(ctx context.Context, tx *sqlx.Tx, c *models.ActivationCode) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE activation_codes
		SET status = ?, used_by_user_id = ?, activated_at = ?, expires_at = ?
		WHERE id = ?`,
		c.Status, c.UsedByUserID, c.ActivatedAt, c.ExpiresAt, c.ID,
	)
	return err
}

func (r *ActivationCodeRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE activation_codes SET status = 'REVOKED', revoked_at = NOW()
		WHERE id = ? AND status = 'UNUSED'`, id)
	return err
}

type ListCodesFilter struct {
	Status  string
	BatchID string
}

func (r *ActivationCodeRepository) List(ctx context.Context, f ListCodesFilter) ([]models.ActivationCode, error) {
	query := `SELECT * FROM activation_codes WHERE 1=1`
	args := []any{}

	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.BatchID != "" {
		query += ` AND batch_id = ?`
		args = append(args, f.BatchID)
	}
	query += ` ORDER BY created_at DESC`

	codes := []models.ActivationCode{}
	err := r.db.SelectContext(ctx, &codes, query, args...)
	return codes, err
}
