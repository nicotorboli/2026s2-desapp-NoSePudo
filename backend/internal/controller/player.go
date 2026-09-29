package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type RestPlayerController struct {
	svc service.PlayerService
}

func NewPlayerController(svc service.PlayerService) *RestPlayerController {
	return &RestPlayerController{svc: svc}
}

func (c *RestPlayerController) GetPlayers() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		filterDTO, err := dto.PlayerFilterDesdeQuery(req)
		if err != nil {
			return httphandler.NewError(http.StatusBadRequest, err.Error(), err)
		}

		result, err := c.svc.ListPlayers(req.Context(), filterDTO)
		if err != nil {
			return err
		}

		return httphandler.Encode(w, http.StatusOK, result)
	}
}

func (c *RestPlayerController) GetPlayerByID() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		idParam := req.PathValue("id")
		if idParam == "" {
			return httphandler.NewError(http.StatusBadRequest, "missing player id", nil)
		}

		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil || id <= 0 {
			return httphandler.NewError(http.StatusBadRequest, "invalid player ID format", err)
		}

		player, err := c.svc.GetPlayerByID(req.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return httphandler.NewError(http.StatusNotFound, "player not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, player)
	}
}
