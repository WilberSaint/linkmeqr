package handlers

import (
	"net/http"

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

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

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
	media, err := h.media.Upload(r.Context(), userID, header.Filename, mimeType, header.Size, file)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "upload_failed", err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, media)
}
