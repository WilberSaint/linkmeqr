package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type LicenseActivationRepository struct {
	db *sqlx.DB
}

func NewLicenseActivationRepository(db *sqlx.DB) *LicenseActivationRepository {
	return &LicenseActivationRepository{db: db}
}

func (r *LicenseActivationRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, a *models.LicenseActivation) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO license_activations
			(id, license_id, activation_code_id, user_id, duration_days_added, previous_expires_at, new_expires_at, activated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.LicenseID, a.ActivationCodeID, a.UserID, a.DurationDaysAdded,
		a.PreviousExpiresAt, a.NewExpiresAt, a.ActivatedAt,
	)
	return err
}
