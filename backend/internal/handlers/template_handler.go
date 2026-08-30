package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
	"linkmeqr/backend/internal/validator"
)

type TemplateHandler struct {
	templates *services.TemplateService
	audit     *services.AuditService
}

func NewTemplateHandler(templates *services.TemplateService, audit *services.AuditService) *TemplateHandler {
	return &TemplateHandler{templates: templates, audit: audit}
}

// templateResponse mirrors models.Template but re-emits DefaultTheme as a
// real JSON object instead of the raw string it's stored as in MySQL —
// the frontend expects `default_theme` to already be a parsed ProfileTheme.
type templateResponse struct {
	ID           string          `json:"id"`
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
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
		DefaultTheme: json.RawMessage(t.DefaultTheme),
		IsActive:     t.IsActive,
		SortOrder:    t.SortOrder,
	}
}

// List handles GET /api/templates (public) — active templates only.
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templates.ListActive(r.Context())
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

// ListAllAdmin handles GET /api/admin/templates (ADMIN) — includes inactive templates.
func (h *TemplateHandler) ListAllAdmin(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templates.ListAll(r.Context())
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

func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.templates.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Template not found.")
		return
	}
	utils.JSON(w, http.StatusOK, toTemplateResponse(*t))
}

type templateRequest struct {
	Slug         string          `json:"slug" validate:"required,min=2,max=60"`
	Name         string          `json:"name" validate:"required,min=2,max=100"`
	Description  *string         `json:"description"`
	DefaultTheme json.RawMessage `json:"default_theme" validate:"required"`
	SortOrder    int             `json:"sort_order"`
}

// Create handles POST /api/admin/templates (ADMIN).
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	t, err := h.templates.Create(r.Context(), req.Slug, req.Name, req.Description, string(req.DefaultTheme), req.SortOrder)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			utils.Error(w, http.StatusConflict, "slug_taken", "A template with this slug already exists.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not create template.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "create_template", "template", t.ID, r.RemoteAddr, map[string]any{"slug": t.Slug})

	utils.JSON(w, http.StatusCreated, toTemplateResponse(*t))
}

// Update handles PATCH /api/admin/templates/:id (ADMIN).
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req templateRequest
	if fields := validator.DecodeAndValidate(r, &req); fields != nil {
		utils.ValidationError(w, fields)
		return
	}

	t, err := h.templates.Get(r.Context(), id)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Template not found.")
		return
	}

	t.Slug = req.Slug
	t.Name = req.Name
	t.Description = req.Description
	t.DefaultTheme = string(req.DefaultTheme)
	t.SortOrder = req.SortOrder

	if err := h.templates.Update(r.Context(), t); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			utils.Error(w, http.StatusConflict, "slug_taken", "A template with this slug already exists.")
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update template.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "update_template", "template", id, r.RemoteAddr, nil)

	utils.JSON(w, http.StatusOK, toTemplateResponse(*t))
}

// Delete handles DELETE /api/admin/templates/:id (ADMIN).
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.templates.Delete(r.Context(), id); err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not delete template.")
		return
	}

	adminID := middleware.UserIDFromContext(r.Context())
	h.audit.Log(r.Context(), adminID, "delete_template", "template", id, r.RemoteAddr, nil)

	utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// SetActive handles POST /api/admin/templates/:id/activate|deactivate (ADMIN).
func (h *TemplateHandler) SetActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := h.templates.SetActive(r.Context(), id, active); err != nil {
			utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not update template status.")
			return
		}

		adminID := middleware.UserIDFromContext(r.Context())
		action := "deactivate_template"
		if active {
			action = "activate_template"
		}
		h.audit.Log(r.Context(), adminID, action, "template", id, r.RemoteAddr, nil)

		utils.JSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
