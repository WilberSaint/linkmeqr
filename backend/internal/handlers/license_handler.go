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
	users      *repository.UserRepository
	audit      *services.AuditService
}

func NewLicenseHandler(licenseSvc *services.LicenseService, licenses *repository.LicenseRepository, users *repository.UserRepository, audit *services.AuditService) *LicenseHandler {
	return &LicenseHandler{licenseSvc: licenseSvc, licenses: licenses, users: users, audit: audit}
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
			utils.Error(w, http.StatusNotFound, "code_not_found", "No encontramos ese código de activación.")
		case errors.Is(err, services.ErrCodeNotUsable):
			utils.Error(w, http.StatusConflict, "code_not_usable", "Este código ya se usó o fue revocado.")
		case errors.Is(err, services.ErrCodeNotYours):
			utils.Error(w, http.StatusForbidden, "code_not_yours", "Este código está reservado para otra cuenta.")
		default:
			utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not activate code.")
		}
		return
	}

	h.audit.Log(r.Context(), userID, "activate_license", "license", license.ID, r.RemoteAddr, nil)
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

// codeResponse mirrors models.ActivationCode but adds the resolved name/email
// of whoever the code is assigned to or was used by, so the admin panel can
// show which client owns each code without a second round trip.
type codeResponse struct {
	models.ActivationCode
	AssignedToName  *string `json:"assigned_to_name"`
	AssignedToEmail *string `json:"assigned_to_email"`
	UsedByName      *string `json:"used_by_name"`
	UsedByEmail     *string `json:"used_by_email"`
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

	idSet := map[string]struct{}{}
	for _, c := range codes {
		if c.AssignedUserID != nil {
			idSet[*c.AssignedUserID] = struct{}{}
		}
		if c.UsedByUserID != nil {
			idSet[*c.UsedByUserID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	names := map[string]models.User{}
	if len(ids) > 0 {
		users, err := h.users.ListByIDs(r.Context(), ids)
		if err == nil {
			for _, u := range users {
				names[u.ID] = u
			}
		}
	}

	out := make([]codeResponse, len(codes))
	for i, c := range codes {
		resp := codeResponse{ActivationCode: c}
		if c.AssignedUserID != nil {
			if u, ok := names[*c.AssignedUserID]; ok {
				resp.AssignedToName, resp.AssignedToEmail = &u.FullName, &u.Email
			}
		}
		if c.UsedByUserID != nil {
			if u, ok := names[*c.UsedByUserID]; ok {
				resp.UsedByName, resp.UsedByEmail = &u.FullName, &u.Email
			}
		}
		out[i] = resp
	}

	utils.JSON(w, http.StatusOK, out)
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

type assignCodeRequest struct {
	// Null releases the reservation and puts the code back in the pool.
	UserID *string `json:"user_id"`
}

// AssignCode handles POST /api/admin/licenses/codes/:id/assign (ADMIN) —
// reserving an unused code for one client so only they can redeem it.
func (h *LicenseHandler) AssignCode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req assignCodeRequest
	if err := validator.Decode(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Cuerpo de la petición inválido.")
		return
	}
	if err := h.licenseSvc.AssignCode(r.Context(), id, req.UserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			utils.Error(w, http.StatusConflict, "code_not_assignable", "Sólo se puede reservar un código que siga sin usar.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo reservar el código.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "assign_activation_code", "activation_code", id, r.RemoteAddr, map[string]any{"user_id": req.UserID})
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
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
