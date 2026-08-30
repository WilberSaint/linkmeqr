package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type PrintCardRepository struct {
	db *sqlx.DB
}

func NewPrintCardRepository(db *sqlx.DB) *PrintCardRepository {
	return &PrintCardRepository{db: db}
}

func (r *PrintCardRepository) ListByUser(ctx context.Context, userID string) ([]models.PrintCard, error) {
	cards := []models.PrintCard{}
	err := r.db.SelectContext(ctx, &cards, `
		SELECT * FROM print_cards WHERE user_id = ? ORDER BY updated_at DESC`, userID)
	return cards, err
}

func (r *PrintCardRepository) GetByID(ctx context.Context, id string) (*models.PrintCard, error) {
	var c models.PrintCard
	err := r.db.GetContext(ctx, &c, `SELECT * FROM print_cards WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByScanCode looks up a card by its short public scan code — the only
// lookup the unauthenticated /q/:code redirect route ever needs, and
// deliberately not by the card's own (longer, UUID) id, which would make
// every printed card's QR encode more data than necessary.
func (r *PrintCardRepository) GetByScanCode(ctx context.Context, code string) (*models.PrintCard, error) {
	var c models.PrintCard
	err := r.db.GetContext(ctx, &c, `SELECT * FROM print_cards WHERE scan_code = ?`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PrintCardRepository) Create(ctx context.Context, c *models.PrintCard) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO print_cards (id, scan_code, user_id, layout_key, title, size_preset, custom_width_cm, custom_height_cm, qr_target_type, qr_target_value, color_overrides, content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ScanCode, c.UserID, c.LayoutKey, c.Title, c.SizePreset, c.CustomWidthCm, c.CustomHeightCm, c.QRTargetType, c.QRTargetValue, c.ColorOverrides, c.Content,
	)
	return err
}

func (r *PrintCardRepository) Update(ctx context.Context, c *models.PrintCard) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE print_cards SET
			layout_key = ?, title = ?, size_preset = ?, custom_width_cm = ?, custom_height_cm = ?, qr_target_type = ?, qr_target_value = ?, color_overrides = ?, content = ?
		WHERE id = ?`,
		c.LayoutKey, c.Title, c.SizePreset, c.CustomWidthCm, c.CustomHeightCm, c.QRTargetType, c.QRTargetValue, c.ColorOverrides, c.Content, c.ID,
	)
	return err
}

func (r *PrintCardRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM print_cards WHERE id = ?`, id)
	return err
}

// UpdateStatus is deliberately separate from Update — moving a card through
// the draft/printed/delivered pipeline is a quick one-off action that
// shouldn't require resending the whole design payload, and shouldn't be
// able to accidentally clobber it either.
func (r *PrintCardRepository) UpdateStatus(ctx context.Context, id string, status models.PrintCardSaleStatus, saleNote *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE print_cards SET status = ?, sale_note = ? WHERE id = ?`, status, saleNote, id)
	return err
}

// SaveLayout writes a card's element tree and bumps its revision counter in
// one transaction with the revision row, so print_cards.layout can never
// disagree with the highest version in print_card_layout_versions — a
// mismatch there would make "restore the previous design" restore the wrong
// thing.
func (r *PrintCardRepository) SaveLayout(ctx context.Context, cardID, layout string, createdBy *string) (int, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// SELECT ... FOR UPDATE so two admins saving the same card concurrently
	// serialize instead of both computing the same next version number and
	// colliding on the unique key.
	var current int
	if err := tx.GetContext(ctx, &current, `SELECT layout_version FROM print_cards WHERE id = ? FOR UPDATE`, cardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	next := current + 1

	if _, err := tx.ExecContext(ctx,
		`UPDATE print_cards SET layout = ?, layout_version = ? WHERE id = ?`, layout, next, cardID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO print_card_layout_versions (id, print_card_id, version, layout, created_by) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), cardID, next, layout, createdBy); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return next, nil
}

// ListLayoutVersions returns a card's revision history, newest first. The
// layout JSON itself is deliberately not selected — a history list only
// needs the metadata, and a few dozen full trees would be a lot of payload
// for a dropdown.
func (r *PrintCardRepository) ListLayoutVersions(ctx context.Context, cardID string) ([]models.PrintCardLayoutRevision, error) {
	rows := []models.PrintCardLayoutRevision{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, print_card_id, version, '' AS layout, created_by, created_at
		FROM print_card_layout_versions
		WHERE print_card_id = ?
		ORDER BY version DESC`, cardID)
	return rows, err
}

// GetLayoutVersion loads one specific revision, including its tree — the
// read behind "restore this version".
func (r *PrintCardRepository) GetLayoutVersion(ctx context.Context, cardID string, version int) (*models.PrintCardLayoutRevision, error) {
	var rev models.PrintCardLayoutRevision
	err := r.db.GetContext(ctx, &rev, `
		SELECT id, print_card_id, version, layout, created_by, created_at
		FROM print_card_layout_versions
		WHERE print_card_id = ? AND version = ?`, cardID, version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}
