package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
)

const maxMemoryMultipart = 8 << 20 // 8 MB

type MediaHandler struct {
	media *services.MediaService
}

func NewMediaHandler(media *services.MediaService) *MediaHandler {
	return &MediaHandler{media: media}
}

func (h *MediaHandler) upload(w http.ResponseWriter, r *http.Request, ownerID string) {
	if err := r.ParseMultipartForm(maxMemoryMultipart); err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Could not parse upload.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "bad_request", "Missing file field.")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	media, err := h.media.Upload(r.Context(), ownerID, header.Filename, mimeType, header.Size, file)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "upload_failed", err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, media)
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, middleware.UserIDFromContext(r.Context()))
}

// UploadForClient handles POST /admin/clients/:id/media/upload — the file
// is attributed to the client, not the admin, so it's owned the same way a
// logo the client uploaded themselves through /me/qr would be.
func (h *MediaHandler) UploadForClient(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, chi.URLParam(r, "id"))
}
