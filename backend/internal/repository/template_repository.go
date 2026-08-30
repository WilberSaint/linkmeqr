package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type TemplateRepository struct {
	db *sqlx.DB
}

func NewTemplateRepository(db *sqlx.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) ListActive(ctx context.Context) ([]models.Template, error) {
	templates := []models.Template{}
	err := r.db.SelectContext(ctx, &templates, `
		SELECT * FROM templates WHERE is_active = 1 ORDER BY sort_order ASC`)
	return templates, err
}

// ListAll returns every template (active or not), for the admin catalog.
func (r *TemplateRepository) ListAll(ctx context.Context) ([]models.Template, error) {
	templates := []models.Template{}
	err := r.db.SelectContext(ctx, &templates, `SELECT * FROM templates ORDER BY sort_order ASC`)
	return templates, err
}

func (r *TemplateRepository) GetByID(ctx context.Context, id string) (*models.Template, error) {
	var t models.Template
	err := r.db.GetContext(ctx, &t, `SELECT * FROM templates WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepository) Create(ctx context.Context, t *models.Template) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO templates (id, slug, name, description, default_theme, is_active, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Slug, t.Name, t.Description, t.DefaultTheme, t.IsActive, t.SortOrder,
	)
	return err
}

func (r *TemplateRepository) Update(ctx context.Context, t *models.Template) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE templates SET
			slug = ?, name = ?, description = ?, default_theme = ?, sort_order = ?
		WHERE id = ?`,
		t.Slug, t.Name, t.Description, t.DefaultTheme, t.SortOrder, t.ID,
	)
	return err
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM templates WHERE id = ?`, id)
	return err
}

func (r *TemplateRepository) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE templates SET is_active = ? WHERE id = ?`, active, id)
	return err
}
