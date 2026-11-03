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
			access:   AccessAnonymous,
			endpoint: s.controllers.Sync.GetPlayerStatistics(),
		},
		{
			pattern:  "POST /sync/{type}/runs",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.CreateRun(),
		},
		{
			pattern:  "GET /sync/runs",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.ListRuns(),
		},
		{
			pattern:  "GET /sync/runs/{id}",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.GetRun(),
		},
		{
			pattern:  "GET /sync/schedules",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.ListSchedules(),
		},
		{
			pattern:  "PUT /sync/schedules/{type}",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.UpdateSchedule(),
		},
		{
			pattern:  "GET /scoring/strategies",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.ListStrategies(),
		},
		{
			pattern:  "GET /scoring/active-strategy",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.GetActiveStrategy(),
		},
		{
			pattern:  "PUT /scoring/active-strategy",
			access:   AccessSuperuser,
			endpoint: s.controllers.Scoring.ActivateStrategy(),
		},
		{
			pattern:  "GET /scoring/league-coefficients/{league}",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.GetLeagueCoefficient(),
		},
		{
			pattern:  "PUT /scoring/league-coefficients/{league}",
			access:   AccessSuperuser,
			endpoint: s.controllers.Scoring.SetLeagueCoefficient(),
		},
		{
			pattern:  "GET /players/{id}/score",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.GetScore(),
		},
		{
			pattern:  "GET /players/{id}/scores",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.GetScoreHistory(),
		},
		{
			pattern:  "GET /players/ranking",
			access:   AccessAnonymous,
			endpoint: s.controllers.Scoring.GetPlayersRanking(),
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
