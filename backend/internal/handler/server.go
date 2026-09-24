package handler

import (
	"log/slog"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/cfg"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func NewServer(
	logger *slog.Logger,
	config *cfg.Config,
	playerSql *dao.PlayerSql,

)
