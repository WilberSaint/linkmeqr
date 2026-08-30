package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

// PrintCardHandler is admin-only: the print-card designer builds cards on
// behalf of a chosen client (so LinkMeQR can produce and sell the physical
// print for them) rather than being a self-service tool clients use on
// their own — every route here takes the client's id from the URL, not
// from the caller's own JWT, and sits under RequireRole("ADMIN").
type PrintCardHandler struct {
	cards     *services.PrintCardService
	profiles  *services.ProfileService
	qr        *services.QRManagementService
	media     *repository.MediaRepository
	mediaSvc  *services.MediaService
	audit     *services.AuditService
	analytics *services.AnalyticsService
}

func NewPrintCardHandler(cards *services.PrintCardService, profiles *services.ProfileService, qr *services.QRManagementService, media *repository.MediaRepository, mediaSvc *services.MediaService, audit *services.AuditService, analytics *services.AnalyticsService) *PrintCardHandler {
	return &PrintCardHandler{cards: cards, profiles: profiles, qr: qr, media: media, mediaSvc: mediaSvc, audit: audit, analytics: analytics}
}

type printCardRequest struct {
	LayoutKey  string  `json:"layout_key" validate:"required,oneof=google_review social_follow menu_scan loyalty_card multi_qr thank_you"`
	Title      *string `json:"title" validate:"omitempty,max=150"`
	SizePreset string  `json:"size_preset" validate:"required,oneof=business_card table_tent sticker_square door_hanger custom"`
	// CustomWidthCm/CustomHeightCm are required together whenever
	// SizePreset is "custom" — enforced in the handler below rather than a
	// struct tag, since validator can't express "required if size_preset ==
	// custom" alongside the other cases staying nil.
	CustomWidthCm  *float64        `json:"custom_width_cm" validate:"omitempty,min=1,max=100"`
	CustomHeightCm *float64        `json:"custom_height_cm" validate:"omitempty,min=1,max=100"`
	QRTargetType   string          `json:"qr_target_type" validate:"required,oneof=profile menu loyalty block custom_url"`
	QRTargetValue  *string         `json:"qr_target_value" validate:"omitempty,max=2048"`
	ColorOverrides json.RawMessage `json:"color_overrides"`
	Content        json.RawMessage `json:"content" validate:"required"`
}

func (req printCardRequest) toInput() services.PrintCardInput {
	var colorOverrides *string
	if len(req.ColorOverrides) > 0 && string(req.ColorOverrides) != "null" {
		s := string(req.ColorOverrides)
		colorOverrides = &s
	}
	return services.PrintCardInput{
		LayoutKey:      models.PrintCardLayout(req.LayoutKey),
		Title:          req.Title,
		SizePreset:     models.PrintCardSizePreset(req.SizePreset),
		CustomWidthCm:  req.CustomWidthCm,
		CustomHeightCm: req.CustomHeightCm,
		QRTargetType:   models.QRTargetType(req.QRTargetType),
		QRTargetValue:  req.QRTargetValue,
		ColorOverrides: colorOverrides,
		Content:        string(req.Content),
	}
}

// previewRequest is printCardRequest plus an optional element tree. When
// Layout is present it wins outright and every template field is ignored —
// the editor is asking "render exactly this", not "render the template I
// started from".
type previewRequest struct {
	printCardRequest
	Layout json.RawMessage `json:"layout"`
}

type printCardResponse struct {
	ID             string          `json:"id"`
	LayoutKey      string          `json:"layout_key"`
	Title          *string         `json:"title"`
	SizePreset     string          `json:"size_preset"`
	CustomWidthCm  *float64        `json:"custom_width_cm"`
	CustomHeightCm *float64        `json:"custom_height_cm"`
	QRTargetType   string          `json:"qr_target_type"`
	QRTargetValue  *string         `json:"qr_target_value"`
	ColorOverrides json.RawMessage `json:"color_overrides"`
	Content        json.RawMessage `json:"content"`
	Status         string          `json:"status"`
	SaleNote       *string         `json:"sale_note"`
	ScanCount      int             `json:"scan_count"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func toPrintCardResponse(c models.PrintCard, scanCount int) printCardResponse {
	var colorOverrides json.RawMessage
	if c.ColorOverrides != nil {
		colorOverrides = json.RawMessage(*c.ColorOverrides)
	}
	return printCardResponse{
		ID:             c.ID,
		LayoutKey:      string(c.LayoutKey),
		Title:          c.Title,
		SizePreset:     string(c.SizePreset),
		CustomWidthCm:  c.CustomWidthCm,
		CustomHeightCm: c.CustomHeightCm,
		QRTargetType:   string(c.QRTargetType),
		QRTargetValue:  c.QRTargetValue,
		ColorOverrides: colorOverrides,
		Content:        json.RawMessage(c.Content),
		Status:         string(c.Status),
		SaleNote:       c.SaleNote,
		ScanCount:      scanCount,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func (h *PrintCardHandler) List(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	cards, err := h.cards.ListMine(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar las tarjetas.")
		return
	}
	scanCounts, err := h.cards.ScanCounts(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar los escaneos.")
		return
	}
	out := make([]printCardResponse, len(cards))
	for i, c := range cards {
		out[i] = toPrintCardResponse(c, scanCounts[c.ID])
	}
	utils.JSON(w, http.StatusOK, out)
}

// QRTargets handles GET /admin/clients/:id/print-cards/qr-targets — the
// list of destinations this specific client's QR can realistically point
// at (their profile/loyalty card always, plus one entry per social/menu/etc.
// block they actually have, plus a custom-URL escape hatch), so the editor
// never offers a destination that doesn't exist for this business.
func (h *PrintCardHandler) QRTargets(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Este cliente no tiene un perfil todavía.")
		return
	}
	targets, err := h.cards.AvailableQRTargets(r.Context(), profile)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar los destinos disponibles.")
		return
	}
	utils.JSON(w, http.StatusOK, targets)
}

// requireCustomSizeIfNeeded catches the one cross-field rule validator's
// struct tags can't express on their own: "custom" needs both dimensions,
// every other preset needs neither.
func requireCustomSizeIfNeeded(w http.ResponseWriter, req printCardRequest) bool {
	if req.SizePreset == "custom" && (req.CustomWidthCm == nil || req.CustomHeightCm == nil) {
		utils.ValidationError(w, map[string]string{"custom_width_cm": "Especifica ancho y alto para un tamaño personalizado."})
		return false
	}
	return true
}

func (h *PrintCardHandler) Create(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	var req printCardRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	if !requireCustomSizeIfNeeded(w, req) {
		return
	}
	c, err := h.cards.Create(r.Context(), clientID, req.toInput())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo crear la tarjeta.")
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "create_print_card", "print_card", c.ID, r.RemoteAddr, map[string]any{"client_id": clientID})
	utils.JSON(w, http.StatusCreated, toPrintCardResponse(*c, 0))
}

func (h *PrintCardHandler) Get(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	c, err := h.cards.Get(r.Context(), clientID, id)
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	scanCount, err := h.cards.ScanCount(r.Context(), c.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar los escaneos.")
		return
	}
	utils.JSON(w, http.StatusOK, toPrintCardResponse(*c, scanCount))
}

func (h *PrintCardHandler) Update(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	var req printCardRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	if !requireCustomSizeIfNeeded(w, req) {
		return
	}
	c, err := h.cards.Update(r.Context(), clientID, id, req.toInput())
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	scanCount, err := h.cards.ScanCount(r.Context(), c.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar los escaneos.")
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "update_print_card", "print_card", c.ID, r.RemoteAddr, map[string]any{"client_id": clientID})
	utils.JSON(w, http.StatusOK, toPrintCardResponse(*c, scanCount))
}

func (h *PrintCardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	if err := h.cards.Delete(r.Context(), clientID, id); err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "delete_print_card", "print_card", id, r.RemoteAddr, map[string]any{"client_id": clientID})
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type updatePrintCardStatusRequest struct {
	Status   string  `json:"status" validate:"required,oneof=draft printed delivered"`
	SaleNote *string `json:"sale_note" validate:"omitempty,max=500"`
}

// UpdateStatus handles PATCH /admin/clients/:id/print-cards/:cardId/status
// — a lightweight action to move a card through draft/printed/delivered
// without resending (or risking overwriting) its whole design.
func (h *PrintCardHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	var req updatePrintCardStatusRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	c, err := h.cards.UpdateStatus(r.Context(), clientID, id, models.PrintCardSaleStatus(req.Status), req.SaleNote)
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	scanCount, err := h.cards.ScanCount(r.Context(), c.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudieron cargar los escaneos.")
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "update_print_card_status", "print_card", c.ID, r.RemoteAddr, map[string]any{"client_id": clientID, "status": req.Status})
	utils.JSON(w, http.StatusOK, toPrintCardResponse(*c, scanCount))
}

func (h *PrintCardHandler) notFoundOrForbidden(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrPrintCardNotOwner) {
		utils.Error(w, http.StatusForbidden, "forbidden", "Esta tarjeta no pertenece a este cliente.")
		return
	}
	utils.Error(w, http.StatusNotFound, "not_found", "Tarjeta no encontrada.")
}

// Export handles GET /admin/clients/:id/print-cards/:cardId/export — SVG
// only (PNG is rasterized client-side from this same SVG, so the backend
// never needs a general SVG rasterizer — see the print-card designer plan).
func (h *PrintCardHandler) Export(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	c, err := h.cards.Get(r.Context(), clientID, id)
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}

	svg, err := h.renderCardSVG(r.Context(), clientID, c, nil, true)
	if err != nil {
		h.renderError(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Disposition", "attachment; filename=\"tarjeta.svg\"")
	_, _ = w.Write([]byte(svg))
}

// Preview handles POST /admin/clients/:id/print-cards/preview — renders SVG
// without persisting anything. It serves two callers: the editor, which
// posts the tree it currently has on screen so every drag shows the real
// exported output; and the "choose a design" picker, which posts only a
// template key and size and gets back that template's seeded tree rendered
// as a thumbnail. Neither one creates a row, so exploring designs never
// litters the card list with abandoned drafts.
func (h *PrintCardHandler) Preview(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	// Decoded before validation because the two accepted shapes have
	// different rules: a tree-only preview has no template fields to
	// validate, and demanding them would reject every save-less preview the
	// editor sends.
	var req previewRequest
	if err := validator.Decode(r, &req); err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Cuerpo de la petición inválido.")
		return
	}

	var layout *models.CardLayout
	draft := &models.PrintCard{}

	if len(req.Layout) > 0 && string(req.Layout) != "null" {
		decoded, err := h.decodeLayout(req.Layout)
		if err != nil {
			utils.ValidationError(w, map[string]string{"layout": "El diseño no tiene un formato válido."})
			return
		}
		layout = decoded
	} else {
		if fields := validator.Validate(&req.printCardRequest); fields != nil {
			utils.ValidationError(w, fields)
			return
		}
		if !requireCustomSizeIfNeeded(w, req.printCardRequest) {
			return
		}
		in := req.toInput()
		draft = &models.PrintCard{
			LayoutKey:      in.LayoutKey,
			Title:          in.Title,
			SizePreset:     in.SizePreset,
			CustomWidthCm:  in.CustomWidthCm,
			CustomHeightCm: in.CustomHeightCm,
			QRTargetType:   in.QRTargetType,
			QRTargetValue:  in.QRTargetValue,
			ColorOverrides: in.ColorOverrides,
			Content:        in.Content,
		}
	}

	svg, err := h.renderCardSVG(r.Context(), clientID, draft, layout, false)
	if err != nil {
		h.renderError(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(svg))
}

// IconPreview handles GET /admin/print-cards/icons/{name}?color=%23fff — one
// built-in glyph, drawn in a 100x100 viewBox so the editor can scale it into
// any element box.
//
// The editor could redraw these glyphs in TypeScript, but then the Google G
// a designer positions would be a different shape from the one that gets
// printed, and every future tweak would have to be made twice. Serving the
// exporter's own artwork keeps the two identical by construction.
func (h *PrintCardHandler) IconPreview(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	color := r.URL.Query().Get("color")
	if color == "" {
		color = "#111827"
	}
	// The color arrives in a URL and is interpolated straight into SVG
	// markup by the glyph helpers, so only accept a literal hex color.
	if !hexColorParam.MatchString(color) {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Color inválido.")
		return
	}

	const box = 100.0
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %g %g">%s</svg>`,
		box, box, services.IconGlyphSVG(name, box/2, box/2, box/2, color),
	)
	// Glyph artwork only changes when the app is redeployed, so it is worth
	// caching hard in the browser: the editor asks for the same handful of
	// icons on every card it opens.
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(svg))
}

var hexColorParam = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

// qrPreviewRequest names one QR destination to render standalone.
type qrPreviewRequest struct {
	TargetType  string  `json:"target_type" validate:"required,oneof=profile menu loyalty block custom_url"`
	TargetValue *string `json:"target_value" validate:"omitempty,max=2048"`
}

// QRPreview handles POST /admin/clients/:id/print-cards/qr-preview — the QR
// for one destination, on its own, in the business's own QR style.
//
// The editor draws the card itself in the browser so dragging is immediate,
// but it cannot generate a QR: encoding one needs the real destination URL
// and the business's saved module/eye styling. Rather than settle for a
// placeholder square (which would let a designer size a QR that turns out
// unscannable in print), the editor fetches the genuine code once per
// destination and caches it.
func (h *PrintCardHandler) QRPreview(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	var req qrPreviewRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Este cliente no tiene un perfil todavía.")
		return
	}
	qrStyle, err := h.qr.GetOrCreate(r.Context(), profile.ID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el estilo del QR.")
		return
	}
	content, err := h.cards.ResolveQRContent(r.Context(), models.QRTargetType(req.TargetType), req.TargetValue, profile)
	if err != nil {
		h.renderError(w, err)
		return
	}
	svg, err := services.RenderSVG(h.qr.ToCustomizationWithContent(r.Context(), qrStyle, content))
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el código QR.")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(svg))
}

// SeedLayout handles POST /admin/clients/:id/print-cards/seed-layout — hands
// the editor a fresh tree for one of the built-in designs at a given size,
// without creating a card. This is what makes the six templates "initial
// state, not render structure": the editor asks for a starting tree and then
// owns it outright.
func (h *PrintCardHandler) SeedLayout(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	var req printCardRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	if !requireCustomSizeIfNeeded(w, req) {
		return
	}
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Este cliente no tiene un perfil todavía.")
		return
	}
	in := req.toInput()
	draft := &models.PrintCard{
		LayoutKey:      in.LayoutKey,
		Title:          in.Title,
		SizePreset:     in.SizePreset,
		CustomWidthCm:  in.CustomWidthCm,
		CustomHeightCm: in.CustomHeightCm,
		QRTargetType:   in.QRTargetType,
		QRTargetValue:  in.QRTargetValue,
		ColorOverrides: in.ColorOverrides,
		Content:        in.Content,
	}
	layout, err := h.cards.SeedLayoutFor(r.Context(), draft, profile)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el diseño inicial.")
		return
	}
	encoded, _ := json.Marshal(layout)
	utils.JSON(w, http.StatusOK, layoutResponse{Layout: encoded, LayoutVersion: 0})
}

func (h *PrintCardHandler) renderError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrNoMenuBlock) {
		utils.Error(w, http.StatusConflict, "no_menu_block", "Este perfil todavía no tiene un bloque de menú configurado.")
		return
	}
	utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo generar la tarjeta.")
}

// renderCardSVG composes a card's full SVG from its element tree. When
// tracked is true and the card has been saved, each QR encodes its own
// short /q/:cardCode[/:slot] link instead of the resolved destination
// directly, so scanning the printed card is counted before the visitor is
// sent on — used for the real Export a client will print, never for the
// live-editing Preview of an unsaved draft, which has no id to track
// against yet.
//
// Nothing about a card's design is decided here any more: the tree says
// what to draw and where, and this function only resolves what the tree
// references (each QR's destination, the logo, uploaded images).
func (h *PrintCardHandler) renderCardSVG(ctx context.Context, clientID string, card *models.PrintCard, layout *models.CardLayout, tracked bool) (string, error) {
	profile, err := h.profiles.GetByUserID(ctx, clientID)
	if err != nil {
		return "", err
	}
	qrStyle, err := h.qr.GetOrCreate(ctx, profile.ID)
	if err != nil {
		return "", err
	}

	if layout == nil {
		layout, err = h.cards.LayoutFor(ctx, card, profile)
		if err != nil {
			return "", err
		}
	}

	assets, err := h.cards.ResolveLayoutAssets(ctx, layout, services.LayoutAssetSource{
		Profile:  profile,
		QRStyle:  qrStyle,
		Tracked:  tracked,
		ScanCode: card.ScanCode,
	}, h.mediaSvc)
	if err != nil {
		return "", err
	}
	return services.RenderLayoutSVG(layout, assets), nil
}

// layoutRequest is the save payload for a card's element tree. BaseVersion
// is the revision the editor had loaded; the service refuses a save built on
// a stale one rather than silently discarding whatever the other admin saved
// in the meantime.
type layoutRequest struct {
	Layout      json.RawMessage `json:"layout" validate:"required"`
	BaseVersion *int            `json:"base_version"`
}

type layoutResponse struct {
	Layout        json.RawMessage `json:"layout"`
	LayoutVersion int             `json:"layout_version"`
}

func (h *PrintCardHandler) decodeLayout(raw json.RawMessage) (*models.CardLayout, error) {
	var layout models.CardLayout
	if err := json.Unmarshal(raw, &layout); err != nil {
		return nil, err
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	return &layout, nil
}

// GetLayout handles GET /admin/clients/:id/print-cards/:cardId/layout — the
// editor's load. A card that predates the tree is seeded on the fly and
// returned at version 0, so opening any card in the new editor works
// whether or not the backfill has run.
func (h *PrintCardHandler) GetLayout(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	card, err := h.cards.Get(r.Context(), clientID, id)
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	profile, err := h.profiles.GetByUserID(r.Context(), clientID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Este cliente no tiene un perfil todavía.")
		return
	}
	layout, err := h.cards.LayoutFor(r.Context(), card, profile)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el diseño.")
		return
	}
	encoded, err := json.Marshal(layout)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "No se pudo cargar el diseño.")
		return
	}
	utils.JSON(w, http.StatusOK, layoutResponse{Layout: encoded, LayoutVersion: card.LayoutVersion})
}

// SaveLayout handles PUT /admin/clients/:id/print-cards/:cardId/layout.
func (h *PrintCardHandler) SaveLayout(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	var req layoutRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}
	layout, err := h.decodeLayout(req.Layout)
	if err != nil {
		utils.ValidationError(w, map[string]string{"layout": "El diseño no tiene un formato válido."})
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	card, err := h.cards.SaveLayout(r.Context(), clientID, id, layout, req.BaseVersion, &adminID)
	if err != nil {
		if errors.Is(err, services.ErrLayoutStale) {
			utils.Error(w, http.StatusConflict, "layout_stale", "Alguien más guardó cambios en esta tarjeta. Recarga para no perder su trabajo.")
			return
		}
		h.notFoundOrForbidden(w, err)
		return
	}
	h.audit.Log(r.Context(), adminID, "update_print_card_layout", "print_card", card.ID, r.RemoteAddr, map[string]any{"client_id": clientID, "version": card.LayoutVersion})

	encoded, _ := json.Marshal(layout)
	utils.JSON(w, http.StatusOK, layoutResponse{Layout: encoded, LayoutVersion: card.LayoutVersion})
}

// ListLayoutVersions handles GET .../layout/versions — the design's revision
// history. Worth having for a product whose output is physically printed:
// one stray drag on a finished card is otherwise unrecoverable.
func (h *PrintCardHandler) ListLayoutVersions(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	versions, err := h.cards.ListLayoutVersions(r.Context(), clientID, id)
	if err != nil {
		h.notFoundOrForbidden(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, versions)
}

// RestoreLayoutVersion handles POST .../layout/versions/:version/restore.
func (h *PrintCardHandler) RestoreLayoutVersion(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")
	id := chi.URLParam(r, "cardId")
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Versión inválida.")
		return
	}
	adminID := middleware.UserIDFromContext(r.Context())
	layout, card, err := h.cards.RestoreLayoutVersion(r.Context(), clientID, id, version, &adminID)
	if err != nil {
		if errors.Is(err, services.ErrNoLayoutVersion) {
			utils.Error(w, http.StatusNotFound, "not_found", "Esa versión del diseño no existe.")
			return
		}
		h.notFoundOrForbidden(w, err)
		return
	}
	h.audit.Log(r.Context(), adminID, "restore_print_card_layout", "print_card", card.ID, r.RemoteAddr, map[string]any{"client_id": clientID, "restored": version})

	encoded, _ := json.Marshal(layout)
	utils.JSON(w, http.StatusOK, layoutResponse{Layout: encoded, LayoutVersion: card.LayoutVersion})
}

// Scan handles GET /q/:code and /q/:code/:slot — the short public link every
// exported card's QR actually encodes. It resolves the card's current
// destination (never a snapshot baked in at export time, so a later edit to
// the card or its target block still redirects correctly), logs the scan,
// and redirects. No auth: this is hit directly by a phone camera.
//
// The slot segment names which QR on the card was scanned. Cards printed
// before the layout refactor encode "left"/"right" (or nothing at all);
// CardLayout.FindQRBySlot matches those against the legacy slots the
// backfill preserved, so paper already in customers' hands keeps working
// even though new exports address QRs by element id.
func (h *PrintCardHandler) Scan(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	slot := chi.URLParam(r, "slot")

	card, err := h.cards.GetByScanCode(r.Context(), code)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	profile, err := h.profiles.GetByUserID(r.Context(), card.UserID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Fall back to the card's own columns whenever the tree cannot answer —
	// an un-backfilled card, or a slot naming an element the designer has
	// since deleted. A printed card must never dead-end.
	targetType, targetValue := card.QRTargetType, card.QRTargetValue
	if layout, err := h.cards.LayoutFor(r.Context(), card, profile); err == nil {
		if el := layout.FindQRBySlot(slot); el != nil {
			var p models.QRProps
			if err := el.DecodeProps(&p); err == nil && p.TargetType != "" {
				targetType, targetValue = p.TargetType, p.TargetValue
			}
		}
	}

	dest, err := h.cards.ResolveQRContent(r.Context(), targetType, targetValue, profile)
	if err != nil {
		// Whatever this card pointed at is gone (e.g. a deleted block) —
		// still worth counting the scan, and sending whoever just scanned a
		// real printed card somewhere useful beats an error page.
		dest = h.qr.ProfileURL(profile.Slug)
	}

	_ = h.analytics.RecordScan(r.Context(), r, profile.ID, card.ID, slot)
	http.Redirect(w, r, dest, http.StatusFound)
}
