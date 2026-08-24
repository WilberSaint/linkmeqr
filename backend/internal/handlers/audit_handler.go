package handlers

import (
	"net/http"

	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

type AuditHandler struct {
	repo *repository.AuditLogRepository
}

func NewAuditHandler(repo *repository.AuditLogRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := repository.ListAuditFilter{
		EntityType: r.URL.Query().Get("entity_type"),
		ActorID:    r.URL.Query().Get("actor_id"),
	}

	logs, err := h.repo.List(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list audit logs.")
		return
	}
	utils.JSON(w, http.StatusOK, logs)
}
