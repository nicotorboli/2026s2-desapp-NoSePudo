package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type PlayerStatisticsService interface {
	GetPlayerStatistics(ctx context.Context, playerID int64) ([]dto.PlayerStatisticsResponse, error)
}

// SyncService es el borde del motor de sincronizaciones programadas: runs y
// schedules. El disparo manual y el cron caen en el mismo Trigger.
type SyncService interface {
	Trigger(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error)
	GetRun(ctx context.Context, id int64) (dto.SyncRunResponse, error)
	ListRuns(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error)
	ListSchedules(ctx context.Context) ([]dto.SyncScheduleResponse, error)
	UpdateSchedule(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error)
}

// RestSyncController reúne la lectura de estadísticas de un jugador con el
// motor de sincronizaciones programadas. Son operaciones relacionadas que
// comparten el mismo borde.
type RestSyncController struct {
	statsSvc PlayerStatisticsService
	syncSvc  SyncService
}

func NewSyncController(statsSvc PlayerStatisticsService, syncSvc SyncService) *RestSyncController {
	return &RestSyncController{
		statsSvc: statsSvc,
		syncSvc:  syncSvc,
	}
}

func (c *RestSyncController) CreateRun() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		typ, err := parseSyncType(req.PathValue("type"))
		if err != nil {
			return err
		}

		actor, err := actorOf(req)
		if err != nil {
			return err
		}
		actorID := actor.ID

		run, err := c.syncSvc.Trigger(req.Context(), typ, &actorID)
		if err != nil {
			return err
		}

		return httphandler.Encode(w, http.StatusAccepted, run)
	}
}

func (c *RestSyncController) GetRun() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		id, err := parseSyncRunID(req.PathValue("id"))
		if err != nil {
			return err
		}

		run, err := c.syncSvc.GetRun(req.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrSyncRunNotFound) {
				return httphandler.NewError(http.StatusNotFound, "sync run not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, run)
	}
}

func (c *RestSyncController) ListRuns() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		query := req.URL.Query()

		offset, err := parseSyncOffset(query.Get("offset"))
		if err != nil {
			return err
		}

		limit, err := parseSyncLimit(query.Get("limit"))
		if err != nil {
			return err
		}

		var typ *model.SyncType
		if raw := query.Get("type"); raw != "" {
			parsed, ok := model.ParseSyncType(raw)
			if !ok {
				return httphandler.NewError(http.StatusBadRequest, "invalid query parameter 'type'", nil)
			}
			typ = &parsed
		}

		page, err := c.syncSvc.ListRuns(req.Context(), typ, offset, limit)
		if err != nil {
			return err
		}

		return httphandler.Encode(w, http.StatusOK, page)
	}
}

func (c *RestSyncController) ListSchedules() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		schedules, err := c.syncSvc.ListSchedules(req.Context())
		if err != nil {
			return err
		}

		return httphandler.Encode(w, http.StatusOK, schedules)
	}
}

func (c *RestSyncController) UpdateSchedule() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		typ, err := parseSyncType(req.PathValue("type"))
		if err != nil {
			return err
		}

		request, err := httphandler.Decode[dto.UpdateSyncScheduleRequest](req)
		if err != nil {
			return err
		}

		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		schedule, err := c.syncSvc.UpdateSchedule(req.Context(), typ, *request.Enabled, request.IntervalMinutes, actor.ID)
		if err != nil {
			if errors.Is(err, service.ErrSyncScheduleNotFound) {
				return httphandler.NewError(http.StatusNotFound, "sync schedule not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, schedule)
	}
}

func (c *RestSyncController) GetPlayerStatistics() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		idParam := req.PathValue("id")
		if idParam == "" {
			return httphandler.NewError(http.StatusBadRequest, "missing player id", nil)
		}

		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil || id <= 0 {
			return httphandler.NewError(http.StatusBadRequest, "invalid player ID format", err)
		}

		stats, err := c.statsSvc.GetPlayerStatistics(req.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return httphandler.NewError(http.StatusNotFound, "player not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, stats)
	}
}

func parseSyncType(raw string) (model.SyncType, error) {
	typ, ok := model.ParseSyncType(raw)
	if !ok {
		return "", httphandler.NewError(http.StatusBadRequest, "invalid sync type", nil)
	}
	return typ, nil
}

func parseSyncRunID(raw string) (int64, error) {
	if raw == "" {
		return 0, httphandler.NewError(http.StatusBadRequest, "missing sync run id", nil)
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, httphandler.NewError(http.StatusBadRequest, "invalid sync run id", err)
	}

	return id, nil
}

func parseSyncOffset(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	offset, err := strconv.Atoi(raw)
	if err != nil || offset < 0 {
		return 0, httphandler.NewError(http.StatusBadRequest, "invalid query parameter 'offset'", err)
	}

	return offset, nil
}

func parseSyncLimit(raw string) (int, error) {
	if raw == "" {
		return 20, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 100 {
		return 0, httphandler.NewError(http.StatusBadRequest, "invalid query parameter 'limit'", err)
	}

	return limit, nil
}
