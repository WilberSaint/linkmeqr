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

type StatsSummary struct {
	repository.Summary
	Timeseries []repository.DailyCount   `json:"timeseries"`
	Devices    []repository.BreakdownRow `json:"devices"`
	OS         []repository.BreakdownRow `json:"os"`
	Browsers   []repository.BreakdownRow `json:"browsers"`
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

	return &StatsSummary{
		Summary:    *summary,
		Timeseries: series,
		Devices:    devices,
		OS:         os,
		Browsers:   browsers,
	}, nil
}
