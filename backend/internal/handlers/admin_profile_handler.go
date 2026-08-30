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

type updateClientLogoRequest struct {
	LogoMediaID *string `json:"logo_media_id"`
	// LogoShape is optional — set together with LogoMediaID whenever the
	// admin just cropped a fresh logo (see ImageCropModal on the frontend),
	// omitted for a plain clear-the-logo call. Shared with the client's own
	// theme.logo_shape (see ProfileHandler.UpdateMyTheme) rather than a
	// print-card-only setting, so a business's logo reads the same way
	// everywhere it appears — its public page and every printed card alike.
	LogoShape *string `json:"logo_shape" validate:"omitempty,oneof=circle rounded square"`
}

// UpdateLogoForClient lets an admin set (or clear) a client's profile logo
// directly from LinkMeQR Studio — the same logo the print-card "Ícono
// superior" draws from — without needing to leave the card editor to find
// it in the client's own profile editor. Deliberately narrower than a
// general profile update: besides the optional logo_shape, it only ever
// touches LogoMediaID, so it can't accidentally clobber the business name,
// slug, or publish state while an admin is just trying to attach a logo.
func (h *AdminProfileHandler) UpdateLogoForClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var req updateClientLogoRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "This client has no profile yet.")
		return
	}

	profile.LogoMediaID = req.LogoMediaID
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update logo.")
		return
	}

	if req.LogoShape != nil {
		if theme, err := h.profiles.GetTheme(r.Context(), profile.ID); err == nil {
			theme.LogoShape = *req.LogoShape
			_ = h.profiles.UpdateTheme(r.Context(), theme)
		}
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "update_client_logo", "profile", profile.ID, r.RemoteAddr, map[string]any{"client_id": clientID})

	utils.JSON(w, http.StatusOK, profile)
}

func (h *AdminProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profiles.List(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list profiles.")
		return
	}
	utils.JSON(w, http.StatusOK, profiles)
}
