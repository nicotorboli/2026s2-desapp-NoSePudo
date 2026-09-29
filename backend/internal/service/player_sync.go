package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/footballdata"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var (
	defaultLeagues       = []string{"PL", "BL1", "PD", "SA", "FL1"}
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

type PlayerSyncService interface {
	SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)
}

type PlayerSyncServiceImpl struct {
	adapter footballdata.Client
	repo    repository.PlayerRepository
	logger  *slog.Logger
}

func NewPlayerSyncService(adapter footballdata.Client, repo repository.PlayerRepository, logger *slog.Logger) *PlayerSyncServiceImpl {
	return &PlayerSyncServiceImpl{
		adapter: adapter,
		repo:    repo,
		logger:  logger,
	}
}

func (s *PlayerSyncServiceImpl) SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
	totalProcessed := 0
	totalUpdated := 0
	totalDeactivated := 0

	for _, code := range defaultLeagues {
		select {
		case <-ctx.Done():
			return dto.SyncPlayersResponse{}, ctx.Err()
		default:
		}

		s.logger.InfoContext(ctx, "Starting sync for league", "league_code", code)

		players, err := s.adapter.FetchLeaguePlayers(ctx, code)
		if err != nil {
			if errors.Is(err, footballdata.ErrRateLimitExceeded) {
				s.logger.WarnContext(ctx, "Rate limit hit while syncing league", "league_code", code, "error", err.Error())
				if totalProcessed > 0 {
					return dto.SyncPlayersResponse{
						Status:           "partial_success",
						Message:          fmt.Sprintf("Sync partially completed; rate limit reached on league %s", code),
						TotalProcessed:   totalProcessed,
						TotalUpdated:     totalUpdated,
						TotalDeactivated: totalDeactivated,
					}, nil
				}
				return dto.SyncPlayersResponse{}, ErrRateLimitExceeded
			}
			s.logger.ErrorContext(ctx, "Failed to fetch league players", "league_code", code, "error", err.Error())
			return dto.SyncPlayersResponse{}, fmt.Errorf("sync league %s: %w", code, err)
		}

		proc, upd, deact, err := s.repo.SavePlayers(ctx, players, code, actor)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to save players to database", "league_code", code, "error", err.Error())
			return dto.SyncPlayersResponse{}, fmt.Errorf("save players for league %s: %w", code, err)
		}

		totalProcessed += proc
		totalUpdated += upd
		totalDeactivated += deact
		s.logger.InfoContext(ctx, "Completed sync for league", "league_code", code, "processed", proc, "updated", upd, "deactivated", deact)
	}

	return dto.SyncPlayersResponse{
		Status:           "success",
		Message:          "Player synchronization completed successfully",
		TotalProcessed:   totalProcessed,
		TotalUpdated:     totalUpdated,
		TotalDeactivated: totalDeactivated,
	}, nil
}
