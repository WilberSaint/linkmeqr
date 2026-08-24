package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type RefreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)`,
		t.ID, t.UserID, t.TokenHash, t.ExpiresAt,
	)
	return err
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	err := r.db.GetContext(ctx, &t, `SELECT * FROM refresh_tokens WHERE token_hash = ?`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, hash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		time.Now().UTC(), hash,
	)
	return err
}

func (r *RefreshTokenRepository) IsValid(t *models.RefreshToken) bool {
	return t.RevokedAt == nil && t.ExpiresAt.After(time.Now().UTC())
}
