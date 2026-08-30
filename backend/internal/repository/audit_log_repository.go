package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

type AuditLogRepository struct {
	db *sqlx.DB
}

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, l *models.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_user_id, action, entity_type, entity_id, metadata, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.ActorUserID, l.Action, l.EntityType, l.EntityID, l.Metadata, l.IPAddress,
	)
	return err
}

type ListAuditFilter struct {
	EntityType string
	ActorID    string
	Action     string
	Limit      int
}

func (r *AuditLogRepository) List(ctx context.Context, f ListAuditFilter) ([]models.AuditLog, error) {
	query := `SELECT * FROM audit_logs WHERE 1=1`
	args := []any{}

	if f.EntityType != "" {
		query += ` AND entity_type = ?`
		args = append(args, f.EntityType)
	}
	if f.ActorID != "" {
		query += ` AND actor_user_id = ?`
		args = append(args, f.ActorID)
	}
	if f.Action != "" {
		query += ` AND action = ?`
		args = append(args, f.Action)
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	logs := []models.AuditLog{}
	err := r.db.SelectContext(ctx, &logs, query, args...)
	return logs, err
}
