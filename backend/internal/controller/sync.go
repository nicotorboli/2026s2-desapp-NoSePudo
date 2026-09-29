package controller

import (
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type RestSyncController struct {
	svc service.PlayerSyncService
}

func NewSyncController(svc service.PlayerSyncService) *RestSyncController {
	return &RestSyncController{svc: svc}
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
