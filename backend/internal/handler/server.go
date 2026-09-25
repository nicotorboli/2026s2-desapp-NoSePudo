package handler

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/cfg"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func NewServer(
	logger *slog.Logger,
	config *cfg.Config,
	controllers *controller.Container,
	playerSql *dao.PlayerSql,
) http.Handler {
	mplex := http.NewServeMux()

	return mplex

}
