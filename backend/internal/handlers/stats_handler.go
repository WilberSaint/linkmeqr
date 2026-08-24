package handlers

import (
	"net/http"
	"strconv"

	"linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/services"
	"linkmeqr/backend/internal/utils"
)

type StatsHandler struct {
	profiles  *services.ProfileService
	analytics *services.AnalyticsService
}

func NewStatsHandler(profiles *services.ProfileService, analytics *services.AnalyticsService) *StatsHandler {
	return &StatsHandler{profiles: profiles, analytics: analytics}
}

func (h *StatsHandler) MySummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	profile, err := h.profiles.GetByUserID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusNotFound, "not_found", "Profile not found.")
		return
	}

	days := 30
	if v := r.URL.Query().Get("range"); v == "7d" {
		days = 7
	} else if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}

	summary, err := h.analytics.FullSummary(r.Context(), profile.ID, days)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal_error", "Could not load stats.")
		return
	}

	utils.JSON(w, http.StatusOK, summary)
}
