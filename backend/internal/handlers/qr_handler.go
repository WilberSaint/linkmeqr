package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type QRHandler struct {
	qr       *services.QRManagementService
	profiles *services.ProfileService
	media    *repository.MediaRepository
}

func NewQRHandler(qr *services.QRManagementService, profiles *services.ProfileService, media *repository.MediaRepository) *QRHandler {
	return &QRHandler{qr: qr, profiles: profiles, media: media}
}

type qrResponse struct {
	models.QRCode
	LogoURL *string `json:"logo_url"`
}

func (h *QRHandler) toResponse(r *http.Request, qr *models.QRCode) qrResponse {
	resp := qrResponse{QRCode: *qr}
	if qr.LogoMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *qr.LogoMediaID); err == nil {
			resp.LogoURL = &m.FilePath
		}
	}
	return resp
}

// myProfile resolves the caller's own profile from their JWT — used by the
// CLIENT-scoped /me/qr* routes.
func (h *QRHandler) myProfile(r *http.Request) (string, string, error) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		return "", "", err
	}
	return profile.ID, profile.Slug, nil
}

// clientProfile resolves the profile of the client named in the URL — used
// by the ADMIN-scoped /admin/clients/:id/qr* routes, so LinkMeQR Studio can
// style a business's QR on their behalf (same underlying qr_codes row the
// client's own /me/qr editor reads and writes).
func (h *QRHandler) clientProfile(r *http.Request) (string, string, error) {
	clientID := chi.URLParam(r, "id")
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		return "", "", err
	}
	return profile.ID, profile.Slug, nil
}

func (h *QRHandler) get(w http.ResponseWriter, r *http.Request, resolve func(*http.Request) (string, string, error)) {
	profileID, _, err := resolve(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}
	utils.JSON(w, http.StatusOK, h.toResponse(r, qr))
}

func (h *QRHandler) Get(w http.ResponseWriter, r *http.Request) { h.get(w, r, h.myProfile) }
func (h *QRHandler) GetForClient(w http.ResponseWriter, r *http.Request) {
	h.get(w, r, h.clientProfile)
}

type updateQRRequest struct {
	ForegroundColor  string  `json:"foreground_color" validate:"required"`
	BackgroundColor  string  `json:"background_color" validate:"required"`
	ModuleStyle      string  `json:"module_style" validate:"required,oneof=square dots rounded"`
	EyeStyle         string  `json:"eye_style" validate:"required,oneof=square circular rounded"`
	LogoMediaID      *string `json:"logo_media_id"`
	LogoStyle        string  `json:"logo_style" validate:"omitempty,oneof=color monochrome dots"`
	EyeColorFromLogo bool    `json:"eye_color_from_logo"`
	PresetIcon       *string `json:"preset_icon" validate:"omitempty,oneof=coffee heart matcha star gift"`
	FrameShape       *string `json:"frame_shape" validate:"omitempty,oneof=custom_logo"`
	ShapeFill        bool    `json:"shape_fill"`
}

func (h *QRHandler) update(w http.ResponseWriter, r *http.Request, resolve func(*http.Request) (string, string, error)) {
	profileID, _, err := resolve(r)
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
	qr.LogoStyle = req.LogoStyle
	if qr.LogoStyle == "" {
		qr.LogoStyle = "color"
	}
	qr.EyeColorFromLogo = req.EyeColorFromLogo
	qr.PresetIcon = req.PresetIcon
	qr.FrameShape = req.FrameShape
	qr.ShapeFill = req.ShapeFill

	if err := h.qr.Update(r.Context(), qr); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update QR settings.")
		return
	}
	utils.JSON(w, http.StatusOK, h.toResponse(r, qr))
}

func (h *QRHandler) Update(w http.ResponseWriter, r *http.Request) { h.update(w, r, h.myProfile) }
func (h *QRHandler) UpdateForClient(w http.ResponseWriter, r *http.Request) {
	h.update(w, r, h.clientProfile)
}

func (h *QRHandler) validate(w http.ResponseWriter, r *http.Request, resolve func(*http.Request) (string, string, error)) {
	profileID, slug, err := resolve(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load QR settings.")
		return
	}

	validation := services.Validate(h.qr.ToCustomization(r.Context(), qr, slug))
	utils.JSON(w, http.StatusOK, validation)
}

func (h *QRHandler) Validate(w http.ResponseWriter, r *http.Request) { h.validate(w, r, h.myProfile) }
func (h *QRHandler) ValidateForClient(w http.ResponseWriter, r *http.Request) {
	h.validate(w, r, h.clientProfile)
}

func (h *QRHandler) export(w http.ResponseWriter, r *http.Request, resolve func(*http.Request) (string, string, error)) {
	profileID, slug, err := resolve(r)
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
	customization := h.qr.ToCustomization(r.Context(), qr, slug)

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

func (h *QRHandler) Export(w http.ResponseWriter, r *http.Request) { h.export(w, r, h.myProfile) }
func (h *QRHandler) ExportForClient(w http.ResponseWriter, r *http.Request) {
	h.export(w, r, h.clientProfile)
}
