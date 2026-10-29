package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type PlayerSyncService interface {
	SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)
}

type PlayerStatisticsSyncService interface {
	SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error)
}

type PlayerStatisticsService interface {
	GetPlayerStatistics(ctx context.Context, playerID int64) ([]dto.PlayerStatisticsResponse, error)
}

// RestSyncController reúne la sincronización del catálogo con football-data.org,
// la sincronización de estadísticas con WhoScored y la lectura de estadísticas
// de un jugador: son operaciones relacionadas y comparten el mismo borde.
type RestSyncController struct {
	svc          PlayerSyncService
	statsSyncSvc PlayerStatisticsSyncService
	statsReadSvc PlayerStatisticsService
}

func NewSyncController(svc PlayerSyncService, statsSyncSvc PlayerStatisticsSyncService, statsReadSvc PlayerStatisticsService) *RestSyncController {
	return &RestSyncController{
		svc:          svc,
		statsSyncSvc: statsSyncSvc,
		statsReadSvc: statsReadSvc,
	}
}

func (c *RestSyncController) SyncPlayers() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		resp, err := c.svc.SyncPlayers(req.Context(), "system/sync")
		if err != nil {
			if errors.Is(err, service.ErrRateLimitExceeded) {
				return httphandler.NewError(http.StatusTooManyRequests, "External provider rate limit hit; existing data is intact", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, resp)
	}
}

func (c *RestSyncController) SyncStatistics() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		resp, err := c.statsSyncSvc.SyncStatistics(req.Context())
		if err != nil {
			if errors.Is(err, service.ErrStatisticsRateLimitExceeded) {
				return httphandler.NewError(http.StatusTooManyRequests, "External provider rate limit hit; existing data is intact", err)
			}
			if errors.Is(err, service.ErrStatisticsProviderBlocked) {
				return httphandler.NewError(http.StatusBadGateway, "External provider blocked the request; existing data is intact", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, resp)
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

		stats, err := c.statsReadSvc.GetPlayerStatistics(req.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return httphandler.NewError(http.StatusNotFound, "player not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, stats)
	}
}
