package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type LicenseHandler struct {
	licenseSvc *services.LicenseService
	licenses   *repository.LicenseRepository
	audit      *services.AuditService
}

func NewLicenseHandler(licenseSvc *services.LicenseService, licenses *repository.LicenseRepository, audit *services.AuditService) *LicenseHandler {
	return &LicenseHandler{licenseSvc: licenseSvc, licenses: licenses, audit: audit}
}

type activateRequest struct {
	Code string `json:"code" validate:"required"`
}

// Activate handles POST /api/me/license/activate (CLIENT).
func (h *LicenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req activateRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	license, err := h.licenseSvc.ActivateCode(r.Context(), userID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCodeNotFound):
			utils.Error(w, http.StatusNotFound, "code_not_found", "Activation code not found.")
		case errors.Is(err, services.ErrCodeNotUsable):
			utils.Error(w, http.StatusConflict, "code_not_usable", "This code has already been used or revoked.")
		default:
			utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not activate code.")
		}
		return
	}

	utils.JSON(w, http.StatusOK, license)
}

// History handles GET /api/me/license/history (CLIENT).
func (h *LicenseHandler) History(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	history, err := h.licenses.ListHistory(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load license history.")
		return
	}
	utils.JSON(w, http.StatusOK, history)
}

// AdminHistory handles GET /api/admin/licenses/:userId/history (ADMIN).
func (h *LicenseHandler) AdminHistory(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	history, err := h.licenses.ListHistory(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load license history.")
		return
	}
	utils.JSON(w, http.StatusOK, history)
}

type generateCodeRequest struct {
	DurationType models.DurationType `json:"duration_type" validate:"required,oneof=1_MONTH 3_MONTHS 6_MONTHS 1_YEAR CUSTOM"`
	CustomDays   int                 `json:"custom_days"`
}

// GenerateCode handles POST /api/admin/licenses/codes (ADMIN).
func (h *LicenseHandler) GenerateCode(w http.ResponseWriter, r *http.Request) {
	var req generateCodeRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	code, err := h.licenseSvc.GenerateCode(r.Context(), adminID, req.DurationType, req.CustomDays)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.audit.Log(r.Context(), adminID, "generate_activation_code", "activation_code", code.ID, r.RemoteAddr, map[string]any{
		"duration_type": code.DurationType,
	})

	utils.JSON(w, http.StatusCreated, code)
}

type generateBatchRequest struct {
	DurationType models.DurationType `json:"duration_type" validate:"required,oneof=1_MONTH 3_MONTHS 6_MONTHS 1_YEAR CUSTOM"`
	CustomDays   int                 `json:"custom_days"`
	Quantity     int                 `json:"quantity" validate:"required,min=1,max=5000"`
}

// GenerateBatch handles POST /api/admin/licenses/codes/batch (ADMIN).
func (h *LicenseHandler) GenerateBatch(w http.ResponseWriter, r *http.Request) {
	var req generateBatchRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	codes, err := h.licenseSvc.GenerateBatch(r.Context(), adminID, req.DurationType, req.CustomDays, req.Quantity)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.audit.Log(r.Context(), adminID, "generate_activation_code_batch", "activation_code_batch", "", r.RemoteAddr, map[string]any{
		"duration_type": req.DurationType,
		"quantity":      req.Quantity,
	})

	utils.JSON(w, http.StatusCreated, codes)
}

// ListCodes handles GET /api/admin/licenses/codes (ADMIN).
func (h *LicenseHandler) ListCodes(w http.ResponseWriter, r *http.Request) {
	filter := repository.ListCodesFilter{
		Status:  r.URL.Query().Get("status"),
		BatchID: r.URL.Query().Get("batch_id"),
	}

	codes, err := h.licenseSvc.ListCodes(r.Context(), filter)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list codes.")
		return
	}
	utils.JSON(w, http.StatusOK, codes)
}

type adminActivateRequest struct {
	DurationType models.DurationType `json:"duration_type" validate:"required,oneof=1_MONTH 3_MONTHS 6_MONTHS 1_YEAR CUSTOM"`
	CustomDays   int                 `json:"custom_days"`
}

// AdminActivate handles POST /api/admin/clients/:id/license/activate (ADMIN).
// Generates a code and activates it for the given client in one step, so the
// admin doesn't have to hop between the Licenses and Clients screens.
func (h *LicenseHandler) AdminActivate(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var req adminActivateRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	license, err := h.licenseSvc.ActivateForClient(r.Context(), adminID, clientID, req.DurationType, req.CustomDays)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.audit.Log(r.Context(), adminID, "admin_activate_client_license", "license", license.ID, r.RemoteAddr, map[string]any{
		"client_id":     clientID,
		"duration_type": req.DurationType,
	})

	utils.JSON(w, http.StatusOK, license)
}

// RevokeCode handles POST /api/admin/licenses/codes/:id/revoke (ADMIN).
func (h *LicenseHandler) RevokeCode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.licenseSvc.RevokeCode(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not revoke code.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "revoke_activation_code", "activation_code", id, r.RemoteAddr, nil)

	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
