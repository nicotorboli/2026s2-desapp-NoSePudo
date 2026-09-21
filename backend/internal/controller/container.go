package controller

import (
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
)

type Controller interface {
	Register(mux *http.ServeMux) any
}

type PlayerController interface {
	Controller
	ListPlayers() []dto.Player
}

type Container struct {
	Player PlayerController
}
