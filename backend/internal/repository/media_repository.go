package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type MediaRepository struct {
	db *sqlx.DB
}

func NewMediaRepository(db *sqlx.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(ctx context.Context, m *models.Media) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO media (id, owner_user_id, file_name, file_path, mime_type, size_bytes, width, height)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.OwnerUserID, m.FileName, m.FilePath, m.MimeType, m.SizeBytes, m.Width, m.Height,
	)
	return err
}

func (r *MediaRepository) GetByID(ctx context.Context, id string) (*models.Media, error) {
	var m models.Media
	err := r.db.GetContext(ctx, &m, `SELECT * FROM media WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
