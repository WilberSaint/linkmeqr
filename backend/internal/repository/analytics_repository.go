package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type AnalyticsRepository struct {
	db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) Create(ctx context.Context, e *models.AnalyticsEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_events
			(id, profile_id, event_type, block_id, device_type, os_name, browser_name, referrer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ProfileID, e.EventType, e.BlockID, e.DeviceType, e.OSName, e.BrowserName, e.Referrer,
	)
	return err
}

type Summary struct {
	TotalViews  int `db:"total_views" json:"total_views"`
	TotalClicks int `db:"total_clicks" json:"total_clicks"`
	Views7d     int `db:"views_7d" json:"views_7d"`
	Views30d    int `db:"views_30d" json:"views_30d"`
}

func (r *AnalyticsRepository) Summary(ctx context.Context, profileID string) (*Summary, error) {
	var s Summary
	err := r.db.GetContext(ctx, &s, `
		SELECT
			(SELECT COUNT(*) FROM analytics_events WHERE profile_id = ? AND event_type = 'VIEW') AS total_views,
			(SELECT COUNT(*) FROM analytics_events WHERE profile_id = ? AND event_type = 'BLOCK_CLICK') AS total_clicks,
			(SELECT COUNT(*) FROM analytics_events WHERE profile_id = ? AND event_type = 'VIEW' AND created_at >= NOW() - INTERVAL 7 DAY) AS views_7d,
			(SELECT COUNT(*) FROM analytics_events WHERE profile_id = ? AND event_type = 'VIEW' AND created_at >= NOW() - INTERVAL 30 DAY) AS views_30d
	`, profileID, profileID, profileID, profileID)
	return &s, err
}

type DailyCount struct {
	Date  string `db:"date" json:"date"`
	Count int    `db:"count" json:"count"`
}

func (r *AnalyticsRepository) Timeseries(ctx context.Context, profileID string, days int) ([]DailyCount, error) {
	rows := []DailyCount{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT DATE(created_at) AS date, COUNT(*) AS count
		FROM analytics_events
		WHERE profile_id = ? AND event_type = 'VIEW' AND created_at >= NOW() - INTERVAL ? DAY
		GROUP BY DATE(created_at)
		ORDER BY date ASC`, profileID, days)
	return rows, err
}

type BreakdownRow struct {
	Label string `db:"label" json:"label"`
	Count int    `db:"count" json:"count"`
}

func (r *AnalyticsRepository) DeviceBreakdown(ctx context.Context, profileID string) ([]BreakdownRow, error) {
	rows := []BreakdownRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT COALESCE(device_type, 'unknown') AS label, COUNT(*) AS count
		FROM analytics_events
		WHERE profile_id = ? AND event_type = 'VIEW'
		GROUP BY device_type
		ORDER BY count DESC`, profileID)
	return rows, err
}

func (r *AnalyticsRepository) OSBreakdown(ctx context.Context, profileID string) ([]BreakdownRow, error) {
	rows := []BreakdownRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT COALESCE(os_name, 'unknown') AS label, COUNT(*) AS count
		FROM analytics_events
		WHERE profile_id = ? AND event_type = 'VIEW'
		GROUP BY os_name
		ORDER BY count DESC`, profileID)
	return rows, err
}

func (r *AnalyticsRepository) BrowserBreakdown(ctx context.Context, profileID string) ([]BreakdownRow, error) {
	rows := []BreakdownRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT COALESCE(browser_name, 'unknown') AS label, COUNT(*) AS count
		FROM analytics_events
		WHERE profile_id = ? AND event_type = 'VIEW'
		GROUP BY browser_name
		ORDER BY count DESC`, profileID)
	return rows, err
}
