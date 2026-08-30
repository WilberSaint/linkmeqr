package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type ClientHandler struct {
	clients     *services.ClientService
	licenses    *repository.LicenseRepository
	audit       *services.AuditService
	jwtSecret   string
	impersonTTL time.Duration
}

func NewClientHandler(clients *services.ClientService, licenses *repository.LicenseRepository, audit *services.AuditService, jwtSecret string, impersonTTL time.Duration) *ClientHandler {
	return &ClientHandler{clients: clients, licenses: licenses, audit: audit, jwtSecret: jwtSecret, impersonTTL: impersonTTL}
}

type clientWithLicense struct {
	models.User
	LicenseStatus        models.LicenseStatus `json:"license_status"`
	LicenseDaysRemaining *int                 `json:"license_days_remaining"`
}

func (h *ClientHandler) withLicense(r *http.Request, u models.User) clientWithLicense {
	license, err := h.licenses.GetByUserID(r.Context(), u.ID)
	if err != nil {
		return clientWithLicense{User: u, LicenseStatus: models.LicenseInactive}
	}
	return clientWithLicense{
		User:                 u,
		LicenseStatus:        services.EffectiveStatus(license),
		LicenseDaysRemaining: services.DaysRemaining(license),
	}
}

type createClientRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	FullName string  `json:"full_name" validate:"required,min=2"`
	Phone    *string `json:"phone"`
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createClientRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	user, err := h.clients.Create(r.Context(), req.Email, req.Password, req.FullName, req.Phone)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			utils.Error(w, http.StatusConflict, "email_taken", "A user with this email already exists.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not create client.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "create_client", "user", user.ID, r.RemoteAddr, map[string]any{"email": user.Email})

	utils.JSON(w, http.StatusCreated, user)
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clients.List(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list clients.")
		return
	}

	out := make([]clientWithLicense, len(clients))
	for i, c := range clients {
		out[i] = h.withLicense(r, c)
	}
	utils.JSON(w, http.StatusOK, out)
}

func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	client, err := h.clients.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Client not found.")
		return
	}
	utils.JSON(w, http.StatusOK, client)
}

type updateClientRequest struct {
	FullName string  `json:"full_name" validate:"required,min=2"`
	Phone    *string `json:"phone"`
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateClientRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	client, err := h.clients.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Client not found.")
		return
	}

	client.FullName = req.FullName
	client.Phone = req.Phone
	if err := h.clients.Update(r.Context(), client); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update client.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "update_client", "user", id, r.RemoteAddr, nil)

	utils.JSON(w, http.StatusOK, client)
}

func (h *ClientHandler) SetActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := h.clients.SetActive(r.Context(), id, active); err != nil {
			utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update client status.")
			return
		}

		adminID := middleware.UserIDFromContext(r.Context())
		action := "deactivate_client"
		if active {
			action = "activate_client"
		}
		h.audit.Log(r.Context(), adminID, action, "user", id, r.RemoteAddr, nil)

		utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

type impersonateResponse struct {
	AccessToken string      `json:"access_token"`
	User        models.User `json:"user"`
}

// Impersonate handles POST /api/admin/clients/:id/impersonate (ADMIN).
// Mints a short-lived, CLIENT-scoped access token so the admin can operate
// the client panel exactly as that client would. No refresh token is issued,
// so the session can only ever last impersonTTL.
func (h *ClientHandler) Impersonate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	client, err := h.clients.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Client not found.")
		return
	}
	if !client.IsActive {
		utils.Error(w, http.StatusConflict, "client_inactive", "Cannot impersonate an inactive client.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())

	token, err := utils.GenerateImpersonationAccessToken(h.jwtSecret, client.ID, string(client.Role), adminID, h.impersonTTL)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not start impersonation session.")
		return
	}

	h.audit.Log(r.Context(), adminID, "impersonate_start", "user", client.ID, r.RemoteAddr, nil)

	utils.JSON(w, http.StatusOK, impersonateResponse{AccessToken: token, User: *client})
}
