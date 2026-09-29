package server

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"

func (s *Server) routes() {
	s.router.Handle("GET /players", httphandler.Wrap(s.controllers.Player.GetPlayers, s.logger))
	s.router.Handle("GET /players/{id}", httphandler.Wrap(s.controllers.Player.GetPlayerByID, s.logger))
	s.router.Handle("POST /players/sync", httphandler.Wrap(s.controllers.Sync.SyncPlayers, s.logger))
}
