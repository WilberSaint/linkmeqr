package services

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

type AnalyticsService struct {
	repo *repository.AnalyticsRepository
}

func NewAnalyticsService(repo *repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) RecordFromRequest(ctx context.Context, r *http.Request, profileID string, eventType models.EventType, blockID *string) error {
	deviceType, osName, browserName := utils.ParseUserAgent(r.UserAgent())
	referrer := r.Referer()

	var referrerPtr *string
	if referrer != "" {
		referrerPtr = &referrer
	}

	event := &models.AnalyticsEvent{
		ID:          uuid.NewString(),
		ProfileID:   profileID,
		EventType:   eventType,
		BlockID:     blockID,
		DeviceType:  &deviceType,
		OSName:      &osName,
		BrowserName: &browserName,
		Referrer:    referrerPtr,
	}
	return s.repo.Create(ctx, event)
}

// RecordScan logs one scan of a print card's tracking QR — separate from
// RecordFromRequest since it carries a print card + slot instead of a block,
// and is called from an unauthenticated public redirect route.
func (s *AnalyticsService) RecordScan(ctx context.Context, r *http.Request, profileID, printCardID, slot string) error {
	deviceType, osName, browserName := utils.ParseUserAgent(r.UserAgent())
	var slotPtr *string
	if slot != "" {
		slotPtr = &slot
	}
	event := &models.AnalyticsEvent{
		ID:          uuid.NewString(),
		ProfileID:   profileID,
		EventType:   models.EventQRScan,
		PrintCardID: &printCardID,
		QRSlot:      slotPtr,
		DeviceType:  &deviceType,
		OSName:      &osName,
		BrowserName: &browserName,
	}
	return s.repo.Create(ctx, event)
}

type StatsSummary struct {
	repository.Summary
	Timeseries  []repository.DailyCount    `json:"timeseries"`
	Devices     []repository.BreakdownRow  `json:"devices"`
	OS          []repository.BreakdownRow  `json:"os"`
	Browsers    []repository.BreakdownRow  `json:"browsers"`
	BlockClicks []repository.BlockClickRow `json:"block_clicks"`
}

func (s *AnalyticsService) FullSummary(ctx context.Context, profileID string, timeseriesDays int) (*StatsSummary, error) {
	summary, err := s.repo.Summary(ctx, profileID)
	if err != nil {
		return nil, err
	}
	series, err := s.repo.Timeseries(ctx, profileID, timeseriesDays)
	if err != nil {
		return nil, err
	}
	devices, err := s.repo.DeviceBreakdown(ctx, profileID)
	if err != nil {
		return nil, err
	}
	os, err := s.repo.OSBreakdown(ctx, profileID)
	if err != nil {
		return nil, err
	}
	browsers, err := s.repo.BrowserBreakdown(ctx, profileID)
	if err != nil {
		return nil, err
	}
	blockClicks, err := s.repo.BlockClicks(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return &StatsSummary{
		Summary:     *summary,
		Timeseries:  series,
		Devices:     devices,
		OS:          os,
		Browsers:    browsers,
		BlockClicks: blockClicks,
	}, nil
}
