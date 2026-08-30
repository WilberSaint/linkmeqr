package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type ProfileThemeRepository struct {
	db *sqlx.DB
}

func NewProfileThemeRepository(db *sqlx.DB) *ProfileThemeRepository {
	return &ProfileThemeRepository{db: db}
}

func (r *ProfileThemeRepository) GetByProfileID(ctx context.Context, profileID string) (*models.ProfileTheme, error) {
	var t models.ProfileTheme
	err := r.db.GetContext(ctx, &t, `SELECT * FROM profile_themes WHERE profile_id = ?`, profileID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ProfileThemeRepository) Create(ctx context.Context, t *models.ProfileTheme) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO profile_themes
			(id, profile_id, background_type, background_value, background_media_id, primary_color,
			 secondary_color, text_color, button_text_color, logo_background_color, logo_text_color,
			 logo_display_mode, logo_shape, font_family, button_style, button_shadow, layout, extra_css_vars)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProfileID, t.BackgroundType, t.BackgroundValue, t.BackgroundMediaID, t.PrimaryColor,
		t.SecondaryColor, t.TextColor, t.ButtonTextColor, t.LogoBackgroundColor, t.LogoTextColor,
		t.LogoDisplayMode, t.LogoShape, t.FontFamily, t.ButtonStyle, t.ButtonShadow, t.Layout, t.ExtraCSSVars,
	)
	return err
}

func (r *ProfileThemeRepository) Update(ctx context.Context, t *models.ProfileTheme) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE profile_themes SET
			background_type = ?, background_value = ?, background_media_id = ?, primary_color = ?,
			secondary_color = ?, text_color = ?, button_text_color = ?, logo_background_color = ?,
			logo_text_color = ?, logo_display_mode = ?, logo_shape = ?, font_family = ?, button_style = ?,
			button_shadow = ?, layout = ?, extra_css_vars = ?
		WHERE profile_id = ?`,
		t.BackgroundType, t.BackgroundValue, t.BackgroundMediaID, t.PrimaryColor,
		t.SecondaryColor, t.TextColor, t.ButtonTextColor, t.LogoBackgroundColor,
		t.LogoTextColor, t.LogoDisplayMode, t.LogoShape, t.FontFamily, t.ButtonStyle,
		t.ButtonShadow, t.Layout, t.ExtraCSSVars, t.ProfileID,
	)
	return err
}
