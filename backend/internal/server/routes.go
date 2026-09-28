package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
)

// route es la declaración de un endpoint: su patrón, el nivel de acceso que
// exige y el endpoint en sí. Los campos van de mayor a menor tamaño porque
// govet corre con fieldalignment.
type route struct {
	endpoint httphandler.Endpoint
	pattern  string
	access   AccessLevel
}

// routes describe las rutas y no registra ninguna.
//
// La forma importa tanto como el chequeo automático. Si esto registrara sobre
// s.router, la manera más fácil de agregar un endpoint sería un s.router.Handle
// más al lado de los otros, y el camino de menor resistencia pasaría de largo
// por la tabla. Devolviendo la descripción no hay ningún mux a mano al que
// colgarle una ruta suelta: saltear la tabla deja de ser lo cómodo y pasa a
// ser algo deliberado.
func (s *Server) routes() []route {
	return []route{
		{
			// FR-015 lo fija anónimo: es la puerta de entrada, y exigir
			// credencial para crear la cuenta que la produce no cerraría.
			pattern:  "POST /auth/register",
			access:   AccessAnonymous,
			endpoint: s.controllers.Auth.Register(),
		},
		{
			// Anónimo por FR-015: es la operación que produce la credencial,
			// así que no puede exigir una.
			pattern:  "POST /auth/login",
			access:   AccessAnonymous,
			endpoint: s.controllers.Auth.Login(),
		},
		{
			pattern: "GET /players",
			// Se declara anónimo porque hoy es la verdad: el middleware de
			// autenticación todavía no existe. FR-015 lo quiere autenticado y
			// pasa a serlo en la misma tarea que trae el middleware, para que
			// la declaración y lo que se aplica nunca digan cosas distintas.
			access:   AccessAnonymous,
			endpoint: s.controllers.Player.GetPlayers(),
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
//
// Entra en pánico en el arranque, y no devuelve un error, porque una ruta mal
// declarada no es una condición que el servidor deba tolerar sirviendo: es un
// error de programación, y la alternativa de arrancar igual es exactamente el
// agujero que esta feature existe para cerrar.
func chainFor(r route, _ *middleware.Container) httphandler.Endpoint {
	switch r.access {
	case AccessAnonymous:
		return r.endpoint
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
