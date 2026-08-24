package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type ProfileBlockRepository struct {
	db *sqlx.DB
}

func NewProfileBlockRepository(db *sqlx.DB) *ProfileBlockRepository {
	return &ProfileBlockRepository{db: db}
}

func (r *ProfileBlockRepository) ListByProfile(ctx context.Context, profileID string) ([]models.ProfileBlock, error) {
	blocks := []models.ProfileBlock{}
	err := r.db.SelectContext(ctx, &blocks, `
		SELECT * FROM profile_blocks WHERE profile_id = ? ORDER BY sort_order ASC`, profileID)
	return blocks, err
}

func (r *ProfileBlockRepository) GetByID(ctx context.Context, id string) (*models.ProfileBlock, error) {
	var b models.ProfileBlock
	err := r.db.GetContext(ctx, &b, `SELECT * FROM profile_blocks WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *ProfileBlockRepository) NextSortOrder(ctx context.Context, profileID string) (int, error) {
	var max sql.NullInt64
	err := r.db.GetContext(ctx, &max, `SELECT MAX(sort_order) FROM profile_blocks WHERE profile_id = ?`, profileID)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64) + 1, nil
}

func (r *ProfileBlockRepository) Create(ctx context.Context, b *models.ProfileBlock) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profile_blocks
			(id, profile_id, block_type, title, description, url, icon, media_id,
			 style_overrides, content, is_visible, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.ProfileID, b.BlockType, b.Title, b.Description, b.URL, b.Icon, b.MediaID,
		b.StyleOverrides, b.Content, b.IsVisible, b.SortOrder,
	)
	return err
}

func (r *ProfileBlockRepository) Update(ctx context.Context, b *models.ProfileBlock) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE profile_blocks SET
			title = ?, description = ?, url = ?, icon = ?, media_id = ?,
			style_overrides = ?, content = ?, is_visible = ?
		WHERE id = ?`,
		b.Title, b.Description, b.URL, b.Icon, b.MediaID,
		b.StyleOverrides, b.Content, b.IsVisible, b.ID,
	)
	return err
}

func (r *ProfileBlockRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM profile_blocks WHERE id = ?`, id)
	return err
}

// Reorder applies a full new ordering for a profile's blocks in one transaction.
func (r *ProfileBlockRepository) Reorder(ctx context.Context, profileID string, order []struct {
	ID        string
	SortOrder int
}) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE profile_blocks SET sort_order = ? WHERE id = ? AND profile_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range order {
		if _, err := stmt.ExecContext(ctx, item.SortOrder, item.ID, profileID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
