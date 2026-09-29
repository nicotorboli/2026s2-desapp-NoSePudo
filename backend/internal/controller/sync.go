package controller

import (
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type SyncController struct {
	svc service.PlayerSyncService
}

func NewSyncController(svc service.PlayerSyncService) *SyncController {
	return &SyncController{svc: svc}
}

func (c *SyncController) SyncPlayers(w http.ResponseWriter, r *http.Request) error {
	resp, err := c.svc.SyncPlayers(r.Context(), "system/sync")
	if err != nil {
		if errors.Is(err, service.ErrRateLimitExceeded) {
			return httphandler.NewRateLimitError("External provider rate limit hit; existing data is intact")
		}
		return err
	}

	return httphandler.Encode(w, http.StatusOK, resp)
}
