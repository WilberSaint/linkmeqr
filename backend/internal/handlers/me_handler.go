package handlers

import (
	"errors"
	"net/http"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
)

type MeHandler struct {
	users    *repository.UserRepository
	licenses *repository.LicenseRepository
}

func NewMeHandler(users *repository.UserRepository, licenses *repository.LicenseRepository) *MeHandler {
	return &MeHandler{users: users, licenses: licenses}
}

type meResponse struct {
	ID       string      `json:"id"`
	Email    string      `json:"email"`
	FullName string      `json:"full_name"`
	Phone    *string     `json:"phone"`
	Role     string      `json:"role"`
	IsActive bool        `json:"is_active"`
	License  licenseInfo `json:"license"`
}

type licenseInfo struct {
	Status        string  `json:"status"`
	ActivatedAt   *string `json:"activated_at"`
	ExpiresAt     *string `json:"expires_at"`
	DaysRemaining *int    `json:"days_remaining"`
}

func (h *MeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "User not found.")
		return
	}

	license, err := h.licenses.GetByUserID(r.Context(), userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load license.")
		return
	}

	info := licenseInfo{Status: string(services.EffectiveStatus(license))}
	if license != nil {
		if license.ActivatedAt != nil {
			s := license.ActivatedAt.Format("2006-01-02T15:04:05Z07:00")
			info.ActivatedAt = &s
		}
		if license.ExpiresAt != nil {
			s := license.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
			info.ExpiresAt = &s
		}
		info.DaysRemaining = services.DaysRemaining(license)
	}

	utils.JSON(w, http.StatusOK, meResponse{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Phone:    user.Phone,
		Role:     string(user.Role),
		IsActive: user.IsActive,
		License:  info,
	})
}
