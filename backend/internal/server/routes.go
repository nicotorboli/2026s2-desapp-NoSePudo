package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// route es la declaración de un endpoint: su patrón, el nivel de acceso que
// exige y el endpoint en sí.
type route struct {
	endpoint httphandler.Endpoint
	pattern  string
	access   AccessLevel
}

// routes describe las rutas y no registra ninguna.

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
			access:   AccessAuthenticated,
			endpoint: s.controllers.Player.GetPlayers(),
		},
		{
			pattern:  "GET /players/{id}",
			access:   AccessAuthenticated,
			endpoint: s.controllers.Player.GetPlayerByID(),
		},
		{
			// Sólo el superusuario: dispara pedidos contra una API externa con
			// rate limit y reescribe el catálogo.
			pattern:  "POST /players/sync",
			access:   AccessSuperuser,
			endpoint: s.controllers.Sync.SyncPlayers(),
		},
	}
}

// buildMux es el único lugar del programa que llama a Handle sobre un
// ServeMux. Recibe la descripción y la convierte en un router, aplicando a
// cada ruta la cadena que su nivel implica.
func buildMux(routes []route, logger *slog.Logger, middlewares *middleware.Container) *http.ServeMux {
	mux := http.NewServeMux()

	for _, r := range routes {
		mux.Handle(r.pattern, httphandler.Wrap(chainFor(r, middlewares), logger))
	}

	return mux
}

// chainFor devuelve el endpoint ya decorado según el nivel declarado.
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
		// Un nivel que exige credencial pero cuyo middleware todavía no se
		// construyó tiene que reventar acá. Servir la ruta sin la cadena sería
		// dejarla abierta mientras la tabla dice que está protegida.
		panic(fmt.Sprintf(
			"la ruta %q declara el nivel %s, pero su middleware todavía no existe",
			r.pattern, r.access,
		))
	}
}
