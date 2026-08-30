package handlers

import (
	"net/http"
	"strconv"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

type AuditHandler struct {
	repo  *repository.AuditLogRepository
	users *repository.UserRepository
}

func NewAuditHandler(repo *repository.AuditLogRepository, users *repository.UserRepository) *AuditHandler {
	return &AuditHandler{repo: repo, users: users}
}

// auditLogResponse adds the actor's resolved name and email to a raw log row.
// The log itself only stores an id; a screen whose whole purpose is answering
// "who did this" cannot make the reader look every id up by hand.
type auditLogResponse struct {
	models.AuditLog
	ActorName  *string `json:"actor_name"`
	ActorEmail *string `json:"actor_email"`
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	filter := repository.ListAuditFilter{
		EntityType: r.URL.Query().Get("entity_type"),
		ActorID:    r.URL.Query().Get("actor_id"),
		Action:     r.URL.Query().Get("action"),
		Limit:      limit,
	}

	logs, err := h.repo.List(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list audit logs.")
		return
	}

	// One lookup per distinct actor rather than per row: a busy log is mostly
	// the same handful of admins repeated.
	ids := map[string]struct{}{}
	for _, l := range logs {
		if l.ActorUserID != nil {
			ids[*l.ActorUserID] = struct{}{}
		}
	}
	actors := make(map[string]models.User, len(ids))
	for id := range ids {
		if u, err := h.users.GetByID(r.Context(), id); err == nil {
			actors[id] = *u
		}
	}

	out := make([]auditLogResponse, len(logs))
	for i, l := range logs {
		out[i] = auditLogResponse{AuditLog: l}
		if l.ActorUserID != nil {
			if u, ok := actors[*l.ActorUserID]; ok {
				name, email := u.FullName, u.Email
				out[i].ActorName, out[i].ActorEmail = &name, &email
			}
		}
	}
	utils.JSON(w, http.StatusOK, out)
}
