package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type LoyaltyHandler struct {
	loyalty  *services.LoyaltyService
	profiles *services.ProfileService
	qr       *services.QRManagementService
	media    *repository.MediaRepository
	audit    *services.AuditService
	wallet   *services.GoogleWalletService
}

func NewLoyaltyHandler(loyalty *services.LoyaltyService, profiles *services.ProfileService, qr *services.QRManagementService, media *repository.MediaRepository, audit *services.AuditService, wallet *services.GoogleWalletService) *LoyaltyHandler {
	return &LoyaltyHandler{loyalty: loyalty, profiles: profiles, qr: qr, media: media, audit: audit, wallet: wallet}
}

// syncWallet best-effort pushes a customer's new stamp count to an
// already-saved Google Wallet pass. Fired off in its own goroutine (with its
// own context, not the request's) so a slow or unreachable Google endpoint
// never adds latency to the stamp/redeem action itself, and errors are only
// logged — most of them are the expected "this customer never saved a pass"
// case, which SyncStamps already treats as a no-op.
func (h *LoyaltyHandler) syncWallet(customerID string, stamps int) {
	if !h.wallet.Enabled() {
		return
	}
	go func() {
		if err := h.wallet.SyncStamps(context.Background(), customerID, stamps); err != nil {
			log.Printf("google wallet sync failed for customer %s: %v", customerID, err)
		}
	}()
}

// branding loads everything the public card needs to look like the
// business's own brand instead of a generic hardcoded style — same fields
// the public profile page already gets via toThemeResponse (profile_handler.go).
func (h *LoyaltyHandler) branding(r *http.Request, userID string) (businessName string, theme *themeResponse, logoURL *string) {
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		return "", nil, nil
	}
	businessName = profile.BusinessName
	if profile.LogoMediaID != nil {
		if m, err := h.media.GetByID(r.Context(), *profile.LogoMediaID); err == nil {
			logoURL = &m.FilePath
		}
	}
	if t, err := h.profiles.GetTheme(r.Context(), profile.ID); err == nil {
		resp := toThemeResponse(r, h.media, t)
		theme = &resp
	}
	return businessName, theme, logoURL
}

// loyaltyCardResponse is the shape returned by both PublicStatus and
// PublicRegister — a single typed struct instead of three near-duplicate
// map literals, so the two states can't silently drift apart on which
// fields they include.
type loyaltyCardResponse struct {
	NeedsRegistration    bool           `json:"needs_registration"`
	BusinessName         string         `json:"business_name"`
	Theme                *themeResponse `json:"theme"`
	LogoURL              *string        `json:"logo_url"`
	FullName             string         `json:"full_name,omitempty"`
	StampsCount          int            `json:"stamps_count,omitempty"`
	StampsRequired       int            `json:"stamps_required"`
	MidRewardStamps      *int           `json:"mid_reward_stamps"`
	MidRewardDescription *string        `json:"mid_reward_description"`
	RewardDescription    *string        `json:"reward_description"`
	JustStamped          bool           `json:"just_stamped,omitempty"`
	IsActive             bool           `json:"is_active"`
}

// loyaltyCookieName is scoped to the BUSINESS (program.UserID), not the
// shareable loyalty_token — deliberately, so "Regenerar enlace" (meant to
// cut off a leaked/abused link from getting NEW registrations) doesn't also
// orphan every customer who already legitimately registered. Their cookie
// still matches the new token's page since it's keyed by the same stable
// business id either way; a customer who never registered still needs the
// current token to register at all, so the leaked-link case is still shut
// down correctly.
func loyaltyCookieName(businessUserID string) string {
	return "lmqr_loy_" + businessUserID
}

func setLoyaltyCookie(w http.ResponseWriter, r *http.Request, businessUserID, identityToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     loyaltyCookieName(businessUserID),
		Value:    identityToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().AddDate(2, 0, 0),
	})
}

// --- Public (no auth) — the "NFC tap" flow ---

// PublicStatus handles GET /api/public/loyalty/:token. If the visitor's
// browser already has an identity cookie for this business, a stamp is
// attempted (subject to the cooldown) and the resulting card status is
// returned. Otherwise it reports that registration is needed.
func (h *LoyaltyHandler) PublicStatus(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	program, err := h.loyalty.ProgramByToken(r.Context(), token)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Programa de lealtad no encontrado.")
		return
	}

	businessName, theme, logoURL := h.branding(r, program.UserID)

	cookie, cookieErr := r.Cookie(loyaltyCookieName(program.UserID))
	if cookieErr != nil || cookie.Value == "" {
		utils.JSON(w, http.StatusOK, loyaltyCardResponse{
			NeedsRegistration:    true,
			BusinessName:         businessName,
			Theme:                theme,
			LogoURL:              logoURL,
			StampsRequired:       program.StampsRequired,
			MidRewardStamps:      program.MidRewardStamps,
			MidRewardDescription: program.MidRewardDescription,
			RewardDescription:    program.RewardDescription,
			IsActive:             program.IsActive,
		})
		return
	}

	c, err := h.loyalty.CustomerByIdentityToken(r.Context(), cookie.Value)
	if err != nil {
		utils.JSON(w, http.StatusOK, loyaltyCardResponse{
			NeedsRegistration:    true,
			BusinessName:         businessName,
			Theme:                theme,
			LogoURL:              logoURL,
			StampsRequired:       program.StampsRequired,
			MidRewardStamps:      program.MidRewardStamps,
			MidRewardDescription: program.MidRewardDescription,
			RewardDescription:    program.RewardDescription,
			IsActive:             program.IsActive,
		})
		return
	}

	stampedCustomer, stamped, err := h.loyalty.StampIfEligible(r.Context(), c, program)
	if err != nil && !errors.Is(err, services.ErrLoyaltyProgramInactive) {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo procesar el sello.")
		return
	}
	if stamped {
		h.syncWallet(stampedCustomer.ID, stampedCustomer.StampsCount)
	}

	utils.JSON(w, http.StatusOK, loyaltyCardResponse{
		NeedsRegistration:    false,
		BusinessName:         businessName,
		Theme:                theme,
		LogoURL:              logoURL,
		FullName:             stampedCustomer.FullName,
		StampsCount:          stampedCustomer.StampsCount,
		StampsRequired:       program.StampsRequired,
		MidRewardStamps:      program.MidRewardStamps,
		MidRewardDescription: program.MidRewardDescription,
		RewardDescription:    program.RewardDescription,
		JustStamped:          stamped,
		IsActive:             program.IsActive,
	})
}

type registerLoyaltyRequest struct {
	FullName string  `json:"full_name" validate:"required,min=2,max=150"`
	Phone    *string `json:"phone"`
}

// PublicRegister handles POST /api/public/loyalty/:token/register — first
// tap: creates the end-customer, counts their first stamp, and sets the
// identity cookie so future taps auto-recognize them.
func (h *LoyaltyHandler) PublicRegister(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	program, err := h.loyalty.ProgramByToken(r.Context(), token)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Programa de lealtad no encontrado.")
		return
	}

	var req registerLoyaltyRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	customer, err := h.loyalty.RegisterCustomer(r.Context(), program, req.FullName, req.Phone)
	if err != nil {
		if errors.Is(err, services.ErrLoyaltyProgramInactive) {
			utils.Error(w, http.StatusConflict, "program_inactive", "Este programa de lealtad no está activo.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo registrar.")
		return
	}

	setLoyaltyCookie(w, r, program.UserID, customer.IdentityToken)

	businessName, theme, logoURL := h.branding(r, program.UserID)
	utils.JSON(w, http.StatusCreated, loyaltyCardResponse{
		BusinessName:         businessName,
		Theme:                theme,
		LogoURL:              logoURL,
		FullName:             customer.FullName,
		StampsCount:          customer.StampsCount,
		StampsRequired:       program.StampsRequired,
		MidRewardStamps:      program.MidRewardStamps,
		MidRewardDescription: program.MidRewardDescription,
		RewardDescription:    program.RewardDescription,
		IsActive:             program.IsActive,
	})
}

// walletSaveResponse always returns 200 with "enabled" telling the frontend
// whether to show the button at all — everything short of "here's your
// link" (not configured, not registered yet, business has no logo) is a
// normal state for this endpoint, not an error condition.
type walletSaveResponse struct {
	Enabled bool    `json:"enabled"`
	SaveURL *string `json:"save_url,omitempty"`
	Reason  *string `json:"reason,omitempty"`
}

// WalletSaveURL handles GET /api/public/loyalty/:token/wallet — returns a
// "Save to Google Wallet" link for whichever customer the request's identity
// cookie identifies. Same no-auth, cookie-based identification as PublicStatus.
func (h *LoyaltyHandler) WalletSaveURL(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	program, err := h.loyalty.ProgramByToken(r.Context(), token)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Programa de lealtad no encontrado.")
		return
	}

	if !h.wallet.Enabled() {
		utils.JSON(w, http.StatusOK, walletSaveResponse{Enabled: false})
		return
	}

	cookie, cookieErr := r.Cookie(loyaltyCookieName(program.UserID))
	if cookieErr != nil || cookie.Value == "" {
		reason := "not_registered"
		utils.JSON(w, http.StatusOK, walletSaveResponse{Enabled: false, Reason: &reason})
		return
	}
	customer, err := h.loyalty.CustomerByIdentityToken(r.Context(), cookie.Value)
	if err != nil {
		reason := "not_registered"
		utils.JSON(w, http.StatusOK, walletSaveResponse{Enabled: false, Reason: &reason})
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), program.UserID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el negocio.")
		return
	}
	if profile.LogoMediaID == nil {
		reason := "no_logo"
		utils.JSON(w, http.StatusOK, walletSaveResponse{Enabled: false, Reason: &reason})
		return
	}
	logoMedia, err := h.media.GetByID(r.Context(), *profile.LogoMediaID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el logo.")
		return
	}

	primaryColor := "#4f46e5"
	if theme, err := h.profiles.GetTheme(r.Context(), profile.ID); err == nil && theme.PrimaryColor != "" {
		primaryColor = theme.PrimaryColor
	}

	saveURL, err := h.wallet.BuildSaveURL(services.LoyaltyWalletInput{
		Program:      program,
		BusinessName: profile.BusinessName,
		LogoURL:      h.qr.AbsoluteURL(logoMedia.FilePath),
		ProfileURL:   h.qr.ProfileURL(profile.Slug),
		LoyaltyURL:   h.qr.LoyaltyURL(program.LoyaltyToken),
		PrimaryColor: primaryColor,
		Customer:     customer,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el pase de Wallet.")
		return
	}
	utils.JSON(w, http.StatusOK, walletSaveResponse{Enabled: true, SaveURL: &saveURL})
}

// --- Client (business owner) — RequireRole("CLIENT") ---

func (h *LoyaltyHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	program, err := h.loyalty.GetOrCreateProgram(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el programa de lealtad.")
		return
	}
	utils.JSON(w, http.StatusOK, program)
}

// ExportQR handles GET /api/me/loyalty/qr?format=png|svg — a QR encoding the
// same URL an NFC tag would be programmed with, styled using the business's
// own saved QR look. No stamping/business logic here at all: this only
// renders an image around a URL the loyalty handlers already serve.
func (h *LoyaltyHandler) ExportQR(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	program, err := h.loyalty.GetOrCreateProgram(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el programa de lealtad.")
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Perfil no encontrado.")
		return
	}

	qr, err := h.qr.GetOrCreate(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el estilo del QR.")
		return
	}

	content := h.qr.LoyaltyURL(program.LoyaltyToken)
	customization := h.qr.ToCustomizationWithContent(r.Context(), qr, content)

	if r.URL.Query().Get("format") == "svg" {
		svg, err := services.RenderSVG(customization)
		if err != nil {
			utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el QR.")
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"lealtad-qr.svg\"")
		_, _ = w.Write([]byte(svg))
		return
	}

	pngBytes, err := services.RenderPNG(customization)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el QR.")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\"lealtad-qr.png\"")
	}
	_, _ = w.Write(pngBytes)
}

type updateLoyaltyRequest struct {
	StampsRequired       int     `json:"stamps_required" validate:"required,min=1,max=100"`
	MidRewardStamps      *int    `json:"mid_reward_stamps" validate:"omitempty,min=1,max=99"`
	MidRewardDescription *string `json:"mid_reward_description" validate:"omitempty,max=255"`
	RewardDescription    *string `json:"reward_description"`
	IsActive             bool    `json:"is_active"`
	RegenerateToken      bool    `json:"regenerate_token"`
}

func (h *LoyaltyHandler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	program, err := h.loyalty.GetOrCreateProgram(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el programa de lealtad.")
		return
	}

	var req updateLoyaltyRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	if req.MidRewardStamps != nil && *req.MidRewardStamps >= req.StampsRequired {
		utils.ValidationError(w, map[string]string{"mid_reward_stamps": "Debe ser menor a los sellos requeridos para el premio final."})
		return
	}

	program.StampsRequired = req.StampsRequired
	program.MidRewardStamps = req.MidRewardStamps
	program.MidRewardDescription = req.MidRewardDescription
	program.RewardDescription = req.RewardDescription
	program.IsActive = req.IsActive

	if req.RegenerateToken {
		if err := h.loyalty.RegenerateToken(r.Context(), program); err != nil {
			utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo regenerar el enlace.")
			return
		}
	} else if err := h.loyalty.UpdateProgram(r.Context(), program); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo actualizar el programa.")
		return
	}

	h.audit.Log(r.Context(), userID, "update_loyalty_program", "loyalty_program", program.ID, r.RemoteAddr, nil)
	utils.JSON(w, http.StatusOK, program)
}

func (h *LoyaltyHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	customers, err := h.loyalty.ListCustomers(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar la lista de clientes.")
		return
	}
	utils.JSON(w, http.StatusOK, customers)
}

func (h *LoyaltyHandler) StampCustomer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	customer, err := h.loyalty.CustomerByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Cliente no encontrado.")
		return
	}

	program, err := h.loyalty.GetOrCreateProgram(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el programa de lealtad.")
		return
	}

	if err := h.loyalty.ManualStamp(r.Context(), userID, customer, program.StampsRequired); err != nil {
		switch {
		case errors.Is(err, services.ErrLoyaltyNotOwner):
			utils.Error(w, http.StatusForbidden, "forbidden", "Este cliente no pertenece a tu negocio.")
		case errors.Is(err, services.ErrLoyaltyCardComplete):
			utils.Error(w, http.StatusConflict, "card_complete", "Este cliente ya completó su tarjeta — canjéala antes de agregar más sellos.")
		default:
			utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo agregar el sello.")
		}
		return
	}
	h.syncWallet(customer.ID, customer.StampsCount)

	h.audit.Log(r.Context(), userID, "loyalty_manual_stamp", "loyalty_customer", id, r.RemoteAddr, nil)
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *LoyaltyHandler) RedeemCustomer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	customer, err := h.loyalty.CustomerByID(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Cliente no encontrado.")
		return
	}

	if err := h.loyalty.Redeem(r.Context(), userID, customer); err != nil {
		if errors.Is(err, services.ErrLoyaltyNotOwner) {
			utils.Error(w, http.StatusForbidden, "forbidden", "Este cliente no pertenece a tu negocio.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo canjear el premio.")
		return
	}
	h.syncWallet(customer.ID, 0)

	h.audit.Log(r.Context(), userID, "loyalty_redeem", "loyalty_customer", id, r.RemoteAddr, nil)
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
