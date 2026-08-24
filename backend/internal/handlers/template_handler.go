package handlers

import (
	"encoding/json"
	"net/http"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

type TemplateHandler struct {
	repo *repository.TemplateRepository
}

func NewTemplateHandler(repo *repository.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{repo: repo}
}

// templateResponse mirrors models.Template but re-emits DefaultTheme as a
// real JSON object instead of the raw string it's stored as in MySQL —
// the frontend expects `default_theme` to already be a parsed ProfileTheme.
type templateResponse struct {
	ID           string          `json:"id"`
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
	PreviewImage *string         `json:"preview_image"`
	DefaultTheme json.RawMessage `json:"default_theme"`
	IsActive     bool            `json:"is_active"`
	SortOrder    int             `json:"sort_order"`
}

func toTemplateResponse(t models.Template) templateResponse {
	return templateResponse{
		ID:           t.ID,
		Slug:         t.Slug,
		Name:         t.Name,
		Description:  t.Description,
		PreviewImage: t.PreviewImage,
		DefaultTheme: json.RawMessage(t.DefaultTheme),
		IsActive:     t.IsActive,
		SortOrder:    t.SortOrder,
	}
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.ListActive(r.Context())
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not list templates.")
		return
	}

	out := make([]templateResponse, len(templates))
	for i, t := range templates {
		out[i] = toTemplateResponse(t)
	}
	utils.JSON(w, http.StatusOK, out)
}
