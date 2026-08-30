package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type QRRepository struct {
	db *sqlx.DB
}

func NewQRRepository(db *sqlx.DB) *QRRepository {
	return &QRRepository{db: db}
}

func (r *QRRepository) GetByProfileID(ctx context.Context, profileID string) (*models.QRCode, error) {
	var q models.QRCode
	err := r.db.GetContext(ctx, &q, `SELECT * FROM qr_codes WHERE profile_id = ?`, profileID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QRRepository) Create(ctx context.Context, q *models.QRCode) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO qr_codes
			(id, profile_id, foreground_color, background_color, module_style, eye_style,
			 logo_media_id, logo_style, eye_color_from_logo, preset_icon, frame_shape, shape_fill, error_correction, has_scannability_warning)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.ID, q.ProfileID, q.ForegroundColor, q.BackgroundColor, q.ModuleStyle, q.EyeStyle,
		q.LogoMediaID, q.LogoStyle, q.EyeColorFromLogo, q.PresetIcon, q.FrameShape, q.ShapeFill, q.ErrorCorrection, q.HasScannabilityWarning,
	)
	return err
}

func (r *QRRepository) Update(ctx context.Context, q *models.QRCode) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE qr_codes SET
			foreground_color = ?, background_color = ?, module_style = ?, eye_style = ?,
			logo_media_id = ?, logo_style = ?, eye_color_from_logo = ?, preset_icon = ?, frame_shape = ?, shape_fill = ?, error_correction = ?, has_scannability_warning = ?
		WHERE profile_id = ?`,
		q.ForegroundColor, q.BackgroundColor, q.ModuleStyle, q.EyeStyle,
		q.LogoMediaID, q.LogoStyle, q.EyeColorFromLogo, q.PresetIcon, q.FrameShape, q.ShapeFill, q.ErrorCorrection, q.HasScannabilityWarning, q.ProfileID,
	)
	return err
}
