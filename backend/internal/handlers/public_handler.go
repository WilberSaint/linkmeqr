package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type PublicHandler struct {
	profiles  *services.ProfileService
	blocks    *services.BlockService
	licenses  *repository.LicenseRepository
	analytics *services.AnalyticsService
	media     *repository.MediaRepository
}

func NewPublicHandler(profiles *services.ProfileService, blocks *services.BlockService, licenses *repository.LicenseRepository, analytics *services.AnalyticsService, media *repository.MediaRepository) *PublicHandler {
	return &PublicHandler{profiles: profiles, blocks: blocks, licenses: licenses, analytics: analytics, media: media}
}

type publicProfileResponse struct {
	Inactive bool                 `json:"inactive"`
	Profile  *publicProfileFields `json:"profile,omitempty"`
	Theme    *themeResponse       `json:"theme,omitempty"`
	Blocks   []blockResponse      `json:"blocks,omitempty"`
}

type publicProfileFields struct {
	Slug         string  `json:"slug"`
	BusinessName string  `json:"business_name"`
	Description  *string `json:"description"`
	LogoURL      *string `json:"logo_url"`
	CoverURL     *string `json:"cover_url"`
}

// GetBySlug handles GET /api/public/profiles/:slug — no auth required.
func (h *PublicHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	profile, err := h.profiles.GetBySlug(r.Context(), slug)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	license, err := h.licenses.GetByUserID(r.Context(), profile.UserID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not check license status.")
		return
	}

	status := services.EffectiveStatus(license)
	if !profile.IsPublished || status != models.LicenseActive {
		utils.JSON(w, http.StatusOK, publicProfileResponse{Inactive: true})
		return
	}

	theme, err := h.profiles.GetTheme(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load theme.")
		return
	}

	blocks, err := h.blocks.List(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load blocks.")
		return
	}

	visible := make([]models.ProfileBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.IsVisible {
			visible = append(visible, b)
		}
	}

	_ = h.analytics.RecordFromRequest(r.Context(), r, profile.ID, models.EventView, nil)

	var logoURL, coverURL *string
	if profile.LogoMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *profile.LogoMediaID); err == nil {
			logoURL = &m.FilePath
		}
	}
	if profile.CoverMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *profile.CoverMediaID); err == nil {
			coverURL = &m.FilePath
		}
	}

	themeResp := toThemeResponse(r, h.media, theme)

	utils.JSON(w, http.StatusOK, publicProfileResponse{
		Inactive: false,
		Profile: &publicProfileFields{
			Slug:         profile.Slug,
			BusinessName: profile.BusinessName,
			Description:  profile.Description,
			LogoURL:      logoURL,
			CoverURL:     coverURL,
		},
		Theme:  &themeResp,
		Blocks: toBlockResponses(r, h.media, visible),
	})
}

type trackEventRequest struct {
	Type    string  `json:"type" validate:"required,oneof=VIEW BLOCK_CLICK"`
	BlockID *string `json:"block_id"`
}

// TrackEvent handles POST /api/public/profiles/:slug/events — no auth required.
func (h *PublicHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	profile, err := h.profiles.GetBySlug(r.Context(), slug)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	var req trackEventRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	if err := h.analytics.RecordFromRequest(r.Context(), r, profile.ID, models.EventType(req.Type), req.BlockID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not record event.")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
