package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

// AdminProfileHandler lets an admin view or provision a client's profile,
// part of the "admin creates client → assigns profile → hands over QR" flow.
type AdminProfileHandler struct {
	profiles *services.ProfileService
	audit    *services.AuditService
}

func NewAdminProfileHandler(profiles *services.ProfileService, audit *services.AuditService) *AdminProfileHandler {
	return &AdminProfileHandler{profiles: profiles, audit: audit}
}

func (h *AdminProfileHandler) GetForClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "This client has no profile yet.")
		return
	}
	utils.JSON(w, http.StatusOK, profile)
}

type createClientProfileRequest struct {
	BusinessName string  `json:"business_name" validate:"required,min=2,max=150"`
	Slug         string  `json:"slug"`
	TemplateID   *string `json:"template_id"`
}

func (h *AdminProfileHandler) CreateForClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var req createClientProfileRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	profile, err := h.profiles.CreateForUser(r.Context(), clientID, req.BusinessName, req.Slug, req.TemplateID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not create profile.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "create_client_profile", "profile", profile.ID, r.RemoteAddr, map[string]any{
		"client_id": clientID,
		"slug":      profile.Slug,
	})

	utils.JSON(w, http.StatusCreated, profile)
}

func (h *AdminProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profiles.List(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list profiles.")
		return
	}
	utils.JSON(w, http.StatusOK, profiles)
}
