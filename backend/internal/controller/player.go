package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type PlayerController struct {
	svc service.PlayerService
}

func NewPlayerController(svc service.PlayerService) *PlayerController {
	return &PlayerController{svc: svc}
}

func (c *PlayerController) GetPlayers(w http.ResponseWriter, r *http.Request) error {
	filterDTO, err := dto.PlayerFilterDesdeQuery(r)
	if err != nil {
		return httphandler.NewBadRequestError(err.Error())
	}

	result, err := c.svc.ListPlayers(r.Context(), filterDTO)
	if err != nil {
		return err
	}

	return httphandler.Encode(w, http.StatusOK, result)
}

func (c *PlayerController) GetPlayerByID(w http.ResponseWriter, r *http.Request) error {
	idParam := r.PathValue("id")
	if idParam == "" {
		return httphandler.NewBadRequestError("missing player id")
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return httphandler.NewBadRequestError("invalid player ID format")
	}

	player, err := c.svc.GetPlayerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return httphandler.NewNotFoundError("player not found")
		}
		return err
	}

	return httphandler.Encode(w, http.StatusOK, player)
}
