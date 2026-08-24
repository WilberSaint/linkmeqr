package services

import (
	"context"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
)

type BlockService struct {
	blocks *repository.ProfileBlockRepository
}

func NewBlockService(blocks *repository.ProfileBlockRepository) *BlockService {
	return &BlockService{blocks: blocks}
}

func (s *BlockService) List(ctx context.Context, profileID string) ([]models.ProfileBlock, error) {
	return s.blocks.ListByProfile(ctx, profileID)
}

type CreateBlockInput struct {
	BlockType      models.BlockType
	Title          *string
	Description    *string
	URL            *string
	Icon           *string
	MediaID        *string
	StyleOverrides *string
	Content        *string
}

func (s *BlockService) Create(ctx context.Context, profileID string, in CreateBlockInput) (*models.ProfileBlock, error) {
	nextOrder, err := s.blocks.NextSortOrder(ctx, profileID)
	if err != nil {
		return nil, err
	}

	block := &models.ProfileBlock{
		ID:             uuid.NewString(),
		ProfileID:      profileID,
		BlockType:      in.BlockType,
		Title:          in.Title,
		Description:    in.Description,
		URL:            in.URL,
		Icon:           in.Icon,
		MediaID:        in.MediaID,
		StyleOverrides: in.StyleOverrides,
		Content:        in.Content,
		IsVisible:      true,
		SortOrder:      nextOrder,
	}
	if err := s.blocks.Create(ctx, block); err != nil {
		return nil, err
	}
	return block, nil
}

func (s *BlockService) Get(ctx context.Context, id string) (*models.ProfileBlock, error) {
	return s.blocks.GetByID(ctx, id)
}

func (s *BlockService) Update(ctx context.Context, b *models.ProfileBlock) error {
	return s.blocks.Update(ctx, b)
}

func (s *BlockService) Delete(ctx context.Context, id string) error {
	return s.blocks.Delete(ctx, id)
}

func (s *BlockService) Duplicate(ctx context.Context, id string) (*models.ProfileBlock, error) {
	original, err := s.blocks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.Create(ctx, original.ProfileID, CreateBlockInput{
		BlockType:      original.BlockType,
		Title:          original.Title,
		Description:    original.Description,
		URL:            original.URL,
		Icon:           original.Icon,
		MediaID:        original.MediaID,
		StyleOverrides: original.StyleOverrides,
		Content:        original.Content,
	})
}

type ReorderItem struct {
	ID        string `json:"id"`
	SortOrder int    `json:"sort_order"`
}

func (s *BlockService) Reorder(ctx context.Context, profileID string, items []ReorderItem) error {
	order := make([]struct {
		ID        string
		SortOrder int
	}, len(items))
	for i, it := range items {
		order[i] = struct {
			ID        string
			SortOrder int
		}{ID: it.ID, SortOrder: it.SortOrder}
	}
	return s.blocks.Reorder(ctx, profileID, order)
}
