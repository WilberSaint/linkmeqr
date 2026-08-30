package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type BlockHandler struct {
	blocks   *services.BlockService
	profiles *services.ProfileService
	media    *repository.MediaRepository
}

func NewBlockHandler(blocks *services.BlockService, profiles *services.ProfileService, media *repository.MediaRepository) *BlockHandler {
	return &BlockHandler{blocks: blocks, profiles: profiles, media: media}
}

// blockResponse mirrors models.ProfileBlock but re-emits StyleOverrides/Content
// as real JSON objects instead of the raw strings they're stored as in MySQL,
// and resolves MediaID into a fetchable MediaURL (e.g. an uploaded menu PDF).
type blockResponse struct {
	ID             string           `json:"id"`
	ProfileID      string           `json:"profile_id"`
	BlockType      models.BlockType `json:"block_type"`
	Title          *string          `json:"title"`
	Description    *string          `json:"description"`
	URL            *string          `json:"url"`
	Icon           *string          `json:"icon"`
	MediaID        *string          `json:"media_id"`
	MediaURL       *string          `json:"media_url"`
	StyleOverrides json.RawMessage  `json:"style_overrides"`
	Content        json.RawMessage  `json:"content"`
	IsVisible      bool             `json:"is_visible"`
	SortOrder      int              `json:"sort_order"`
}

func toBlockResponse(r *http.Request, media *repository.MediaRepository, b *models.ProfileBlock) blockResponse {
	resp := blockResponse{
		ID:          b.ID,
		ProfileID:   b.ProfileID,
		BlockType:   b.BlockType,
		Title:       b.Title,
		Description: b.Description,
		URL:         b.URL,
		Icon:        b.Icon,
		MediaID:     b.MediaID,
		IsVisible:   b.IsVisible,
		SortOrder:   b.SortOrder,
	}
	if b.StyleOverrides != nil {
		resp.StyleOverrides = json.RawMessage(*b.StyleOverrides)
	}
	if b.Content != nil {
		resp.Content = json.RawMessage(*b.Content)
	}
	if b.MediaID != nil {
		if m, err := media.GetByID(r.Context(), *b.MediaID); err == nil {
			resp.MediaURL = &m.FilePath
		}
	}
	return resp
}

func toBlockResponses(r *http.Request, media *repository.MediaRepository, blocks []models.ProfileBlock) []blockResponse {
	out := make([]blockResponse, len(blocks))
	for i, b := range blocks {
		out[i] = toBlockResponse(r, media, &b)
	}
	return out
}

func (h *BlockHandler) myProfileID(r *http.Request) (string, error) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		return "", err
	}
	return profile.ID, nil
}

func (h *BlockHandler) List(w http.ResponseWriter, r *http.Request) {
	profileID, err := h.myProfileID(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	blocks, err := h.blocks.List(r.Context(), profileID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list blocks.")
		return
	}
	utils.JSON(w, http.StatusOK, toBlockResponses(r, h.media, blocks))
}

type createBlockRequest struct {
	BlockType      models.BlockType `json:"block_type" validate:"required"`
	Title          *string          `json:"title"`
	Description    *string          `json:"description"`
	URL            *string          `json:"url"`
	Icon           *string          `json:"icon"`
	MediaID        *string          `json:"media_id"`
	StyleOverrides json.RawMessage  `json:"style_overrides"`
	Content        json.RawMessage  `json:"content"`
}

func rawToStringPtr(raw json.RawMessage) *string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	s := string(raw)
	return &s
}

func (h *BlockHandler) Create(w http.ResponseWriter, r *http.Request) {
	profileID, err := h.myProfileID(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	var req createBlockRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	block, err := h.blocks.Create(r.Context(), profileID, services.CreateBlockInput{
		BlockType:      req.BlockType,
		Title:          req.Title,
		Description:    req.Description,
		URL:            services.NormalizeBlockURL(req.BlockType, req.URL),
		Icon:           req.Icon,
		MediaID:        req.MediaID,
		StyleOverrides: rawToStringPtr(req.StyleOverrides),
		Content:        rawToStringPtr(req.Content),
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not create block.")
		return
	}
	utils.JSON(w, http.StatusCreated, toBlockResponse(r, h.media, block))
}

func (h *BlockHandler) verifyOwnership(r *http.Request, blockID string) bool {
	profileID, err := h.myProfileID(r)
	if err != nil {
		return false
	}
	block, err := h.blocks.Get(r.Context(), blockID)
	if err != nil {
		return false
	}
	return block.ProfileID == profileID
}

type updateBlockRequest struct {
	Title          *string         `json:"title"`
	Description    *string         `json:"description"`
	URL            *string         `json:"url"`
	Icon           *string         `json:"icon"`
	MediaID        *string         `json:"media_id"`
	StyleOverrides json.RawMessage `json:"style_overrides"`
	Content        json.RawMessage `json:"content"`
	IsVisible      *bool           `json:"is_visible"`
}

func (h *BlockHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.verifyOwnership(r, id) {
		utils.Error(w, http.StatusForbidden, "forbidden", "You do not own this block.")
		return
	}

	var req updateBlockRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	block, err := h.blocks.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Block not found.")
		return
	}

	block.Title = req.Title
	block.Description = req.Description
	block.URL = services.NormalizeBlockURL(block.BlockType, req.URL)
	block.Icon = req.Icon
	block.MediaID = req.MediaID
	block.StyleOverrides = rawToStringPtr(req.StyleOverrides)
	block.Content = rawToStringPtr(req.Content)
	if req.IsVisible != nil {
		block.IsVisible = *req.IsVisible
	}

	if err := h.blocks.Update(r.Context(), block); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update block.")
		return
	}
	utils.JSON(w, http.StatusOK, toBlockResponse(r, h.media, block))
}

func (h *BlockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.verifyOwnership(r, id) {
		utils.Error(w, http.StatusForbidden, "forbidden", "You do not own this block.")
		return
	}

	if err := h.blocks.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not delete block.")
		return
	}
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *BlockHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.verifyOwnership(r, id) {
		utils.Error(w, http.StatusForbidden, "forbidden", "You do not own this block.")
		return
	}

	block, err := h.blocks.Duplicate(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not duplicate block.")
		return
	}
	utils.JSON(w, http.StatusCreated, toBlockResponse(r, h.media, block))
}

type reorderRequest struct {
	Items []services.ReorderItem `json:"items" validate:"required,min=1"`
}

func (h *BlockHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	profileID, err := h.myProfileID(r)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	var req reorderRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	if err := h.blocks.Reorder(r.Context(), profileID, req.Items); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not reorder blocks.")
		return
	}
	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
