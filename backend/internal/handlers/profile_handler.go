package handlers

import (
	"net/http"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type ProfileHandler struct {
	profiles *services.ProfileService
	media    *repository.MediaRepository
}

func NewProfileHandler(profiles *services.ProfileService, media *repository.MediaRepository) *ProfileHandler {
	return &ProfileHandler{profiles: profiles, media: media}
}

type profileResponse struct {
	models.Profile
	LogoURL  *string `json:"logo_url"`
	CoverURL *string `json:"cover_url"`
}

func (h *ProfileHandler) toResponse(r *http.Request, p *models.Profile) profileResponse {
	resp := profileResponse{Profile: *p}
	if p.LogoMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *p.LogoMediaID); err == nil {
			resp.LogoURL = &m.FilePath
		}
	}
	if p.CoverMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *p.CoverMediaID); err == nil {
			resp.CoverURL = &m.FilePath
		}
	}
	return resp
}

func (h *ProfileHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}
	utils.JSON(w, http.StatusOK, h.toResponse(r, profile))
}

type updateProfileRequest struct {
	BusinessName string  `json:"business_name" validate:"required,min=2,max=150"`
	Description  *string `json:"description"`
	TemplateID   *string `json:"template_id"`
	LogoMediaID  *string `json:"logo_media_id"`
	IsPublished  bool    `json:"is_published"`
}

func (h *ProfileHandler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req updateProfileRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	profile.BusinessName = req.BusinessName
	profile.Description = req.Description
	profile.TemplateID = req.TemplateID
	profile.LogoMediaID = req.LogoMediaID
	profile.IsPublished = req.IsPublished

	if err := h.profiles.Update(r.Context(), profile); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update profile.")
		return
	}

	utils.JSON(w, http.StatusOK, h.toResponse(r, profile))
}

type themeResponse struct {
	models.ProfileTheme
	BackgroundURL *string `json:"background_url"`
}

func toThemeResponse(r *http.Request, media *repository.MediaRepository, t *models.ProfileTheme) themeResponse {
	resp := themeResponse{ProfileTheme: *t}
	if t.BackgroundMediaID != nil {
		if m, err := media.GetByID(r.Context(), *t.BackgroundMediaID); err == nil {
			resp.BackgroundURL = &m.FilePath
		}
	}
	return resp
}

func (h *ProfileHandler) GetMyTheme(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	theme, err := h.profiles.GetTheme(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Theme not found.")
		return
	}
	utils.JSON(w, http.StatusOK, toThemeResponse(r, h.media, theme))
}

type updateThemeRequest struct {
	BackgroundType      string  `json:"background_type" validate:"required,oneof=color gradient pattern image"`
	BackgroundValue     string  `json:"background_value" validate:"required"`
	BackgroundMediaID   *string `json:"background_media_id"`
	PrimaryColor        string  `json:"primary_color" validate:"required"`
	SecondaryColor      string  `json:"secondary_color" validate:"required"`
	TextColor           string  `json:"text_color" validate:"required"`
	ButtonTextColor     string  `json:"button_text_color" validate:"required"`
	LogoBackgroundColor string  `json:"logo_background_color" validate:"required"`
	LogoTextColor       string  `json:"logo_text_color" validate:"required"`
	LogoDisplayMode     string  `json:"logo_display_mode" validate:"required,oneof=image initial"`
	LogoShape           string  `json:"logo_shape" validate:"required,oneof=circle rounded square"`
	FontFamily          string  `json:"font_family" validate:"required"`
	ButtonStyle         string  `json:"button_style" validate:"required,oneof=rounded square pill outline"`
	ButtonShadow        bool    `json:"button_shadow"`
	Layout              string  `json:"layout" validate:"required,oneof=list grid"`
}

func (h *ProfileHandler) UpdateMyTheme(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	var req updateThemeRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	theme, err := h.profiles.GetTheme(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Theme not found.")
		return
	}

	theme.BackgroundType = req.BackgroundType
	theme.BackgroundValue = req.BackgroundValue
	theme.BackgroundMediaID = req.BackgroundMediaID
	theme.PrimaryColor = req.PrimaryColor
	theme.SecondaryColor = req.SecondaryColor
	theme.TextColor = req.TextColor
	theme.ButtonTextColor = req.ButtonTextColor
	theme.LogoBackgroundColor = req.LogoBackgroundColor
	theme.LogoTextColor = req.LogoTextColor
	theme.LogoDisplayMode = req.LogoDisplayMode
	theme.LogoShape = req.LogoShape
	theme.FontFamily = req.FontFamily
	theme.ButtonStyle = req.ButtonStyle
	theme.ButtonShadow = req.ButtonShadow
	theme.Layout = req.Layout

	if err := h.profiles.UpdateTheme(r.Context(), theme); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update theme.")
		return
	}

	utils.JSON(w, http.StatusOK, toThemeResponse(r, h.media, theme))
}
