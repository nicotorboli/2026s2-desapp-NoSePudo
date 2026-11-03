package dto

import (
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// SyncRunProgress es el avance efímero de un run en curso. No se persiste: el
// service lo arma en memoria y lo descarta cuando el run termina.
type SyncRunProgress struct {
	Message string `json:"message"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
}

type SyncRunResponse struct {
	TriggeredBy      *int64           `json:"triggeredBy"`
	FinishedAt       *string          `json:"finishedAt"`
	Progress         *SyncRunProgress `json:"progress"`
	Type             string           `json:"type"`
	Trigger          string           `json:"trigger"`
	Status           string           `json:"status"`
	Message          string           `json:"message"`
	Error            string           `json:"error"`
	StartedAt        string           `json:"startedAt"`
	ID               int64            `json:"id"`
	TotalProcessed   int              `json:"totalProcessed"`
	TotalUpdated     int              `json:"totalUpdated"`
	TotalDeactivated int              `json:"totalDeactivated"`
	TotalMatched     int              `json:"totalMatched"`
	TotalUnmatched   int              `json:"totalUnmatched"`
	TotalInserted    int              `json:"totalInserted"`
}

type SyncRunPageResponse struct {
	Items  []SyncRunResponse `json:"items"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
	Total  int               `json:"total"`
}

type SyncScheduleResponse struct {
	UpdatedBy       *int64 `json:"updatedBy"`
	UpdatedAt       string `json:"updatedAt"`
	NextRunAt       string `json:"nextRunAt"`
	Type            string `json:"type"`
	IntervalMinutes int    `json:"intervalMinutes"`
	Enabled         bool   `json:"enabled"`
}

type UpdateSyncScheduleRequest struct {
	Enabled         *bool `json:"enabled"`
	IntervalMinutes int   `json:"intervalMinutes"`
}

func (r *UpdateSyncScheduleRequest) Validate() error {
	if r.Enabled == nil {
		return errors.New(`field "enabled" is required`)
	}
	if r.IntervalMinutes < model.MinSyncIntervalMinutes || r.IntervalMinutes > model.MaxSyncIntervalMinutes {
		return fmt.Errorf(
			"field %q must be between %d and %d",
			"intervalMinutes",
			model.MinSyncIntervalMinutes,
			model.MaxSyncIntervalMinutes,
		)
	}
	return nil
}

func SyncRunDesdeModelo(run model.SyncRun, progress *SyncRunProgress) SyncRunResponse {
	response := SyncRunResponse{
		StartedAt:        run.StartedAt.Format(time.RFC3339),
		Type:             string(run.Type),
		Trigger:          string(run.Trigger),
		Status:           string(run.Status),
		Message:          run.Message,
		Error:            run.Error,
		Progress:         progress,
		ID:               run.ID,
		TriggeredBy:      run.TriggeredBy,
		TotalProcessed:   run.TotalProcessed,
		TotalUpdated:     run.TotalUpdated,
		TotalDeactivated: run.TotalDeactivated,
		TotalMatched:     run.TotalMatched,
		TotalUnmatched:   run.TotalUnmatched,
		TotalInserted:    run.TotalInserted,
	}
	if run.FinishedAt != nil {
		finished := run.FinishedAt.Format(time.RFC3339)
		response.FinishedAt = &finished
	}
	return response
}

func SyncScheduleDesdeModelo(schedule model.SyncSchedule) SyncScheduleResponse {
	return SyncScheduleResponse{
		UpdatedAt:       schedule.UpdatedAt.Format(time.RFC3339),
		NextRunAt:       schedule.NextRunAt.Format(time.RFC3339),
		Type:            string(schedule.Type),
		IntervalMinutes: schedule.IntervalMinutes,
		UpdatedBy:       schedule.UpdatedBy,
		Enabled:         schedule.Enabled,
	}
}
