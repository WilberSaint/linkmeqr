package services

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

type AuditService struct {
	repo *repository.AuditLogRepository
}

func NewAuditService(repo *repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(ctx context.Context, actorUserID, action, entityType, entityID, ip string, metadata map[string]any) {
	var metaPtr *string
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			m := string(b)
			metaPtr = &m
		}
	}

	var actorPtr *string
	if actorUserID != "" {
		actorPtr = &actorUserID
	}
	var entityPtr *string
	if entityID != "" {
		entityPtr = &entityID
	}
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}

	log := &models.AuditLog{
		ID:          uuid.NewString(),
		ActorUserID: actorPtr,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityPtr,
		Metadata:    metaPtr,
		IPAddress:   ipPtr,
	}

	// Best-effort: audit logging must never block or fail the primary request.
	_ = s.repo.Create(ctx, log)
}
