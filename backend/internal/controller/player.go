package controller

import (
	"fmt"
	"io"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
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

func (c *RestPlayerController) ListPlayers(w http.ResponseWriter, req *http.Request) {
	players := c.svc.ListPlayers()
	playersDto := make([]dto.Player, len(players))
	for i := range players {
		playersDto[i] = dto.FromModel(players[i])
		io.WriteString(w, fmt.Sprintf("%v", playersDto[i]))
	}

}

func (c *RestPlayerController) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /player/", c.ListPlayers)
}
