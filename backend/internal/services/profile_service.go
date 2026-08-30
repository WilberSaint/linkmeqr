package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

var ErrSlugTaken = errors.New("slug already in use")

var slugSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

type ProfileService struct {
	profiles *repository.ProfileRepository
	themes   *repository.ProfileThemeRepository
}

func NewProfileService(profiles *repository.ProfileRepository, themes *repository.ProfileThemeRepository) *ProfileService {
	return &ProfileService{profiles: profiles, themes: themes}
}

func Slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, " ", "-")
	s = slugSanitizer.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "negocio"
	}
	return s
}

// CreateForUser provisions a profile + default theme for a client user,
// picking a unique slug derived from the requested one (or the business name).
func (s *ProfileService) CreateForUser(ctx context.Context, userID, businessName, requestedSlug string, templateID *string) (*models.Profile, error) {
	base := Slugify(requestedSlug)
	if base == "" || requestedSlug == "" {
		base = Slugify(businessName)
	}

	slug := base
	for i := 1; ; i++ {
		exists, err := s.profiles.SlugExists(ctx, slug)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	profile := &models.Profile{
		ID:           uuid.NewString(),
		UserID:       userID,
		Slug:         slug,
		BusinessName: businessName,
		TemplateID:   templateID,
		IsPublished:  true,
	}
	if err := s.profiles.Create(ctx, profile); err != nil {
		return nil, err
	}

	theme := &models.ProfileTheme{
		ID:                  uuid.NewString(),
		ProfileID:           profile.ID,
		BackgroundType:      "color",
		BackgroundValue:     "#ffffff",
		PrimaryColor:        "#111827",
		SecondaryColor:      "#6366f1",
		TextColor:           "#111827",
		ButtonTextColor:     "#ffffff",
		LogoBackgroundColor: "#111827",
		LogoTextColor:       "#ffffff",
		LogoDisplayMode:     "initial",
		LogoShape:           "circle",
		FontFamily:          "Inter",
		ButtonStyle:         "rounded",
		Layout:              "list",
	}
	if err := s.themes.Create(ctx, theme); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *ProfileService) GetByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	return s.profiles.GetByUserID(ctx, userID)
}

func (s *ProfileService) GetBySlug(ctx context.Context, slug string) (*models.Profile, error) {
	return s.profiles.GetBySlug(ctx, slug)
}

func (s *ProfileService) Update(ctx context.Context, p *models.Profile) error {
	return s.profiles.Update(ctx, p)
}

func (s *ProfileService) GetTheme(ctx context.Context, profileID string) (*models.ProfileTheme, error) {
	return s.themes.GetByProfileID(ctx, profileID)
}

func (s *ProfileService) UpdateTheme(ctx context.Context, t *models.ProfileTheme) error {
	return s.themes.Update(ctx, t)
}

func (s *ProfileService) List(ctx context.Context) ([]models.Profile, error) {
	return s.profiles.List(ctx)
}
