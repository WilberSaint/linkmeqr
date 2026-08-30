package services

import (
	"context"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

type TemplateService struct {
	repo *repository.TemplateRepository
}

func NewTemplateService(repo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) ListActive(ctx context.Context) ([]models.Template, error) {
	return s.repo.ListActive(ctx)
}

func (s *TemplateService) ListAll(ctx context.Context) ([]models.Template, error) {
	return s.repo.ListAll(ctx)
}

func (s *TemplateService) Get(ctx context.Context, id string) (*models.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) Create(ctx context.Context, slug, name string, description *string, defaultTheme string, sortOrder int) (*models.Template, error) {
	t := &models.Template{
		ID:           uuid.NewString(),
		Slug:         slug,
		Name:         name,
		Description:  description,
		DefaultTheme: defaultTheme,
		IsActive:     true,
		SortOrder:    sortOrder,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TemplateService) Update(ctx context.Context, t *models.Template) error {
	return s.repo.Update(ctx, t)
}

func (s *TemplateService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) SetActive(ctx context.Context, id string, active bool) error {
	return s.repo.SetActive(ctx, id, active)
}
