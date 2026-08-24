package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type LicenseRepository struct {
	db *sqlx.DB
}

func NewLicenseRepository(db *sqlx.DB) *LicenseRepository {
	return &LicenseRepository{db: db}
}

func (r *LicenseRepository) GetByUserID(ctx context.Context, userID string) (*models.License, error) {
	var l models.License
	err := r.db.GetContext(ctx, &l, `SELECT * FROM licenses WHERE user_id = ?`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByUserIDForUpdate must run inside a transaction; it locks the row (or
// signals that none exists yet) so concurrent activations can't race.
func (r *LicenseRepository) GetByUserIDForUpdate(ctx context.Context, tx *sqlx.Tx, userID string) (*models.License, error) {
	var l models.License
	err := tx.GetContext(ctx, &l, `SELECT * FROM licenses WHERE user_id = ? FOR UPDATE`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LicenseRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, l *models.License) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO licenses (id, user_id, status, activated_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.UserID, l.Status, l.ActivatedAt, l.ExpiresAt,
	)
	return err
}

func (r *LicenseRepository) UpdateTx(ctx context.Context, tx *sqlx.Tx, l *models.License) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE licenses SET status = ?, activated_at = ?, expires_at = ? WHERE id = ?`,
		l.Status, l.ActivatedAt, l.ExpiresAt, l.ID,
	)
	return err
}

func (r *LicenseRepository) ListHistory(ctx context.Context, userID string) ([]models.LicenseActivation, error) {
	activations := []models.LicenseActivation{}
	err := r.db.SelectContext(ctx, &activations, `
		SELECT * FROM license_activations WHERE user_id = ? ORDER BY activated_at DESC`, userID)
	return activations, err
}
