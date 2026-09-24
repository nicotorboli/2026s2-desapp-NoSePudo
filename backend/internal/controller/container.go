package controller

import (
	"net/http"
)

type Controller interface {
	Register(mux *http.ServeMux)
}

type PlayerController interface {
	Controller
	ListPlayers(http.ResponseWriter, *http.Request)
}

type Container struct {
	Player PlayerController
}

func NewContainer(p PlayerController) *Container {
	return &Container{
		Player: p,
	}
}
