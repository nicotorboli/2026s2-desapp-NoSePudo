package controller

import (
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type RestStatisticsSyncController struct {
	svc service.PlayerStatisticsSyncService
}

func NewStatisticsSyncController(svc service.PlayerStatisticsSyncService) *RestStatisticsSyncController {
	return &RestStatisticsSyncController{svc: svc}
}

func (c *RestStatisticsSyncController) SyncStatistics() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		resp, err := c.svc.SyncStatistics(req.Context())
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
