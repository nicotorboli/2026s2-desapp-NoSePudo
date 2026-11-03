package service

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerRepository interface {
	ListPlayers(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error)
	GetPlayerByID(ctx context.Context, id int64) (model.Player, error)
	SavePlayers(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error)
}
