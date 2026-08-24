package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type ProfileRepository struct {
	db *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	var p models.Profile
	err := r.db.GetContext(ctx, &p, `SELECT * FROM profiles WHERE user_id = ?`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) GetBySlug(ctx context.Context, slug string) (*models.Profile, error) {
	var p models.Profile
	err := r.db.GetContext(ctx, &p, `SELECT * FROM profiles WHERE slug = ?`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) GetByID(ctx context.Context, id string) (*models.Profile, error) {
	var p models.Profile
	err := r.db.GetContext(ctx, &p, `SELECT * FROM profiles WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM profiles WHERE slug = ?`, slug)
	return count > 0, err
}

func (r *ProfileRepository) Create(ctx context.Context, p *models.Profile) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profiles (id, user_id, slug, business_name, description, template_id, is_published)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Slug, p.BusinessName, p.Description, p.TemplateID, p.IsPublished,
	)
	return err
}

func (r *ProfileRepository) Update(ctx context.Context, p *models.Profile) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE profiles SET
			business_name = ?, description = ?, logo_media_id = ?, cover_media_id = ?,
			template_id = ?, is_published = ?
		WHERE id = ?`,
		p.BusinessName, p.Description, p.LogoMediaID, p.CoverMediaID,
		p.TemplateID, p.IsPublished, p.ID,
	)
	return err
}

func (r *ProfileRepository) List(ctx context.Context) ([]models.Profile, error) {
	profiles := []models.Profile{}
	err := r.db.SelectContext(ctx, &profiles, `SELECT * FROM profiles ORDER BY created_at DESC`)
	return profiles, err
}
