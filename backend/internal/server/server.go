package server

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func NewServer(
	logger *slog.Logger,
	cfg *configuration.Cfg,
	controllers *controller.Container,
	playerSql *dao.PlayerSql,
) http.Handler {
	mux := http.NewServeMux()

	addRoutes(mux, logger, controllers, nil)

	return mux
}
