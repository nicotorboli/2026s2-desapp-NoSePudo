package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type PlayerStatisticsService interface {
	GetPlayerStatistics(ctx context.Context, playerID int64) ([]dto.PlayerStatisticsResponse, error)
}

type PlayerStatisticsServiceImpl struct {
	players    repository.PlayerRepository
	statistics repository.PlayerStatisticsRepository
}

func NewPlayerStatisticsService(players repository.PlayerRepository, statistics repository.PlayerStatisticsRepository) *PlayerStatisticsServiceImpl {
	return &PlayerStatisticsServiceImpl{
		players:    players,
		statistics: statistics,
	}
}

func (s *PlayerStatisticsServiceImpl) GetPlayerStatistics(ctx context.Context, playerID int64) ([]dto.PlayerStatisticsResponse, error) {
	if _, err := s.players.GetPlayerByID(ctx, playerID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rows, err := s.statistics.GetPlayerStatistics(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("service get player statistics: %w", err)
	}

	items := make([]dto.PlayerStatisticsResponse, len(rows))
	for i, row := range rows {
		items[i] = dto.PlayerStatisticsDesdeModelo(row)
	}

	return items, nil
}
