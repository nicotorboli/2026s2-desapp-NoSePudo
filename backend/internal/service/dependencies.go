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

type PlayerStatisticsRepository interface {
	SaveStatistics(ctx context.Context, rows []model.PlayerStatistics) (int, int, error)
	GetPlayerStatistics(ctx context.Context, playerID int64) ([]model.PlayerStatistics, error)
}

type FootballDataClient interface {
	FetchLeaguePlayers(ctx context.Context, leagueCode string) ([]model.Player, error)
}

type WhoscoredClient interface {
	FetchLeagueStatistics(ctx context.Context, leagueCode string) ([]model.PlayerStatisticsRow, error)
}
