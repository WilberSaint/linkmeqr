package handlers

import (
	"errors"
	"net/http"

	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         any    `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	pair, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.Error(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
			return
		}
		if errors.Is(err, services.ErrAccountInactive) {
			utils.Error(w, http.StatusForbidden, "account_inactive", "This account has been deactivated.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not process login.")
		return
	}

	utils.JSON(w, http.StatusOK, tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         pair.User,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh token is invalid or expired.")
		return
	}

	utils.JSON(w, http.StatusOK, tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         pair.User,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	_ = h.auth.Logout(r.Context(), req.RefreshToken)
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
