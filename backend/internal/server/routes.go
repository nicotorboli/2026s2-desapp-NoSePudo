package server

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"

func (s *Server) routes() {
	s.router.Handle("GET /players", httphandler.Wrap(s.controllers.Player.GetPlayers(), s.logger))
}
