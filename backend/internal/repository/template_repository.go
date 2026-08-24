package repository

import (
	"context"

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
