package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

var allowedMimeTypes = map[string]string{
	"image/png":       ".png",
	"image/jpeg":      ".jpg",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

const maxUploadSize = 15 << 20 // 15 MB (covers restaurant menu PDFs)

type MediaService struct {
	repo        *repository.MediaRepository
	storagePath string
}

func NewMediaService(repo *repository.MediaRepository, storagePath string) *MediaService {
	return &MediaService{repo: repo, storagePath: storagePath}
}

func (s *MediaService) Upload(ctx context.Context, ownerUserID string, fileName, mimeType string, size int64, content io.Reader) (*models.Media, error) {
	ext, ok := allowedMimeTypes[mimeType]
	if !ok {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}
	if size > maxUploadSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", maxUploadSize)
	}

	if err := os.MkdirAll(s.storagePath, 0o755); err != nil {
		return nil, fmt.Errorf("prepare storage dir: %w", err)
	}

	id := uuid.NewString()
	storedName := id + ext
	fullPath := filepath.Join(s.storagePath, storedName)

	limited := io.LimitReader(content, maxUploadSize+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(buf)) > maxUploadSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", maxUploadSize)
	}

	if err := os.WriteFile(fullPath, buf, 0o644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	var width, height *int
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(buf)); err == nil {
		w, h := cfg.Width, cfg.Height
		width, height = &w, &h
	}

	media := &models.Media{
		ID:          id,
		OwnerUserID: ownerUserID,
		FileName:    fileName,
		FilePath:    "/media/" + storedName,
		MimeType:    mimeType,
		SizeBytes:   int64(len(buf)),
		Width:       width,
		Height:      height,
	}
	if err := s.repo.Create(ctx, media); err != nil {
		return nil, err
	}
	return media, nil
}
