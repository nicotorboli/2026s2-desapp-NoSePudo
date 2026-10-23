package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type PlayerController interface {
	GetPlayers() httphandler.Endpoint
	GetPlayerByID() httphandler.Endpoint
}

type SyncController interface {
	SyncPlayers() httphandler.Endpoint
}

type AuthController interface {
	Register() httphandler.Endpoint
	Login() httphandler.Endpoint
	Refresh() httphandler.Endpoint
	Logout() httphandler.Endpoint
}

type MarketController interface {
	PlaceBuyOrder() httphandler.Endpoint
	PlaceSellOrder() httphandler.Endpoint
	ListOpenOrders() httphandler.Endpoint
	CancelOrder() httphandler.Endpoint
	Portfolio() httphandler.Endpoint
	Transactions() httphandler.Endpoint
}

type Container struct {
	Player PlayerController
	Sync   SyncController
	Auth   AuthController
	Market MarketController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
		Sync:   NewSyncController(services.Sync),
		Auth:   NewAuthController(services.Auth),
		Market: NewMarketController(services.Market, services.Portfolio),
	}
}
