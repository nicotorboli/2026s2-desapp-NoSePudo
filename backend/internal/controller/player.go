package controller

import (
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerService interface {
	ListPlayers() []model.Player
}

type RestPlayerController struct {
	svc PlayerService
}

func NewPlayerController(svc PlayerService) *RestPlayerController {
	return &RestPlayerController{
		svc: svc,
	}
}

func (c *RestPlayerController) GetPlayers() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		players := c.svc.ListPlayers()
		playersDto := make([]dto.Player, len(players))
		for i := range players {
			playersDto[i] = dto.FromModel(players[i])
		}

		if err := httphandler.Encode(w, http.StatusOK, playersDto); err != nil {
			return err
		}

		return nil
	}
}
