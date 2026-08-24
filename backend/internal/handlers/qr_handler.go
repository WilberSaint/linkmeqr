package handlers

import (
	"net/http"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type QRHandler struct {
	qr       *services.QRManagementService
	profiles *services.ProfileService
}

func NewQRHandler(qr *services.QRManagementService, profiles *services.ProfileService) *QRHandler {
	return &QRHandler{qr: qr, profiles: profiles}
}

func (h *QRHandler) myProfile(r *http.Request) (string, string, error) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		return "", "", err
	}
	return profile.ID, profile.Slug, nil
}

func (h *QRHandler) Get(w http.ResponseWriter, r *http.Request) {
	profileID, _, err := h.myProfile(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}
	utils.JSON(w, http.StatusOK, qr)
}

type updateQRRequest struct {
	ForegroundColor string  `json:"foreground_color" validate:"required"`
	BackgroundColor string  `json:"background_color" validate:"required"`
	ModuleStyle     string  `json:"module_style" validate:"required,oneof=square dots rounded"`
	EyeStyle        string  `json:"eye_style" validate:"required,oneof=square circular rounded"`
	LogoMediaID     *string `json:"logo_media_id"`
}

func (h *QRHandler) Update(w http.ResponseWriter, r *http.Request) {
	profileID, _, err := h.myProfile(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	var req updateQRRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}

	qr.ForegroundColor = req.ForegroundColor
	qr.BackgroundColor = req.BackgroundColor
	qr.ModuleStyle = req.ModuleStyle
	qr.EyeStyle = req.EyeStyle
	qr.LogoMediaID = req.LogoMediaID

	if err := h.qr.Update(r.Context(), qr); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update QR settings.")
		return
	}
	utils.JSON(w, http.StatusOK, qr)
}

func (h *QRHandler) Validate(w http.ResponseWriter, r *http.Request) {
	profileID, slug, err := h.myProfile(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}

	validation := services.Validate(h.qr.ToCustomization(qr, slug))
	utils.JSON(w, http.StatusOK, validation)
}

func (h *QRHandler) Export(w http.ResponseWriter, r *http.Request) {
	profileID, slug, err := h.myProfile(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}

	format := r.URL.Query().Get("format")
	customization := h.qr.ToCustomization(qr, slug)

	if format == "svg" {
		svg, err := services.RenderSVG(customization)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not render QR.")
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"qr-"+slug+".svg\"")
		_, _ = w.Write([]byte(svg))
		return
	}

	pngBytes, err := services.RenderPNG(customization)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not render QR.")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\"qr-"+slug+".png\"")
	}
	_, _ = w.Write(pngBytes)
}
