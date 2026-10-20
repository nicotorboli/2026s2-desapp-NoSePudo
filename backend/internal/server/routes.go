package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type route struct {
	endpoint httphandler.Endpoint
	pattern  string
	access   AccessLevel
}

func (s *Server) routes() []route {
	return []route{
		{
			pattern:  "POST /auth/register",
			access:   AccessAnonymous,
			endpoint: s.controllers.Auth.Register(),
		},
		{
			pattern:  "POST /auth/login",
			access:   AccessAnonymous,
			endpoint: s.controllers.Auth.Login(),
		},
		{
			pattern:  "POST /auth/refresh",
			access:   AccessRenewal,
			endpoint: s.controllers.Auth.Refresh(),
		},
		{
			pattern:  "POST /auth/logout",
			access:   AccessAuthenticated,
			endpoint: s.controllers.Auth.Logout(),
		},
		{
			pattern:  "GET /players",
			access:   AccessAnonymous,
			endpoint: s.controllers.Player.GetPlayers(),
		},
		{
			pattern:  "GET /players/{id}",
			access:   AccessAnonymous,
			endpoint: s.controllers.Player.GetPlayerByID(),
		},
		{
			pattern:  "GET /players/{id}/statistics",
			access:   AccessAuthenticated,
			endpoint: s.controllers.Sync.GetPlayerStatistics(),
		},
		{
			pattern:  "POST /players/sync",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.SyncPlayers(),
		},
		{
			pattern:  "POST /players/statistics/sync",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.SyncStatistics(),
		},
	}
}

func buildMux(routes []route, logger *slog.Logger, middlewares *middleware.Container) *http.ServeMux {
	mux := http.NewServeMux()

	for _, r := range routes {
		mux.Handle(r.pattern, httphandler.Wrap(chainFor(r, middlewares), logger))
	}

	return mux
}

func chainFor(r route, middlewares *middleware.Container) httphandler.Endpoint {
	switch r.access {
	case AccessAnonymous:
		return r.endpoint
	case AccessAuthenticated:
		return middlewares.Authentication.RequireAccessToken()(r.endpoint)
	case AccessRenewal:
		return middlewares.Authentication.RequireRefreshToken()(r.endpoint)
	case AccessSuperuser:
		return middlewares.Authentication.RequireAccessToken()(
			middlewares.Authorization.Require(model.PrivilegeSuperuser)(r.endpoint),
		)
	case AccessUndeclared:
		panic(fmt.Sprintf("la ruta %q no declara su nivel de acceso", r.pattern))
	default:
		panic(fmt.Sprintf(
			"la ruta %q declara el nivel %s, pero su middleware todavía no existe",
			r.pattern, r.access,
		))
	}
}
