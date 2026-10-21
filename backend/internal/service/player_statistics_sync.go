package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/whoscored"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var (
	ErrStatisticsRateLimitExceeded = errors.New("statistics provider rate limit exceeded")
	ErrStatisticsProviderBlocked   = errors.New("statistics provider blocked the request")
)

const maxPlayersPerLeague = 10000

type PlayerStatisticsSyncService interface {
	SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error)
}

type PlayerStatisticsSyncServiceImpl struct {
	adapter    whoscored.Client
	players    repository.PlayerRepository
	statistics repository.PlayerStatisticsRepository
	logger     *slog.Logger
}

func NewPlayerStatisticsSyncService(adapter whoscored.Client, players repository.PlayerRepository, statistics repository.PlayerStatisticsRepository, logger *slog.Logger) *PlayerStatisticsSyncServiceImpl {
	return &PlayerStatisticsSyncServiceImpl{
		adapter:    adapter,
		players:    players,
		statistics: statistics,
		logger:     logger,
	}
}

func (s *PlayerStatisticsSyncServiceImpl) SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error) {
	totalProcessed := 0
	totalMatched := 0
	totalUnmatched := 0
	totalInserted := 0

	for _, code := range defaultLeagues {
		select {
		case <-ctx.Done():
			return dto.SyncStatisticsResponse{}, ctx.Err()
		default:
		}

		s.logger.InfoContext(ctx, "Starting statistics sync for league", "league_code", code)

		proc, matched, unmatched, inserted, err := s.syncLeague(ctx, code)
		if err != nil {
			if errors.Is(err, ErrStatisticsRateLimitExceeded) || errors.Is(err, ErrStatisticsProviderBlocked) {
				if totalProcessed > 0 {
					return dto.SyncStatisticsResponse{
						Status:         "partial_success",
						Message:        fmt.Sprintf("Statistics sync partially completed; provider failed on league %s", code),
						TotalProcessed: totalProcessed,
						TotalMatched:   totalMatched,
						TotalUnmatched: totalUnmatched,
						TotalInserted:  totalInserted,
					}, nil
				}
				return dto.SyncStatisticsResponse{}, err
			}
			s.logger.ErrorContext(ctx, "Failed to sync statistics for league", "league_code", code, "error", err.Error())
			return dto.SyncStatisticsResponse{}, fmt.Errorf("sync statistics league %s: %w", code, err)
		}

		totalProcessed += proc
		totalMatched += matched
		totalUnmatched += unmatched
		totalInserted += inserted
		s.logger.InfoContext(ctx, "Completed statistics sync for league", "league_code", code, "processed", proc, "matched", matched, "unmatched", unmatched, "inserted", inserted)
	}

	return dto.SyncStatisticsResponse{
		Status:         "success",
		Message:        "Player statistics synchronization completed successfully",
		TotalProcessed: totalProcessed,
		TotalMatched:   totalMatched,
		TotalUnmatched: totalUnmatched,
		TotalInserted:  totalInserted,
	}, nil
}

func (s *PlayerStatisticsSyncServiceImpl) syncLeague(ctx context.Context, code string) (int, int, int, int, error) {
	rows, err := s.adapter.FetchLeagueStatistics(ctx, code)
	if err != nil {
		if errors.Is(err, whoscored.ErrRateLimitExceeded) {
			return 0, 0, 0, 0, ErrStatisticsRateLimitExceeded
		}
		if errors.Is(err, whoscored.ErrBotChallenge) {
			return 0, 0, 0, 0, ErrStatisticsProviderBlocked
		}
		return 0, 0, 0, 0, fmt.Errorf("fetch statistics for league %s: %w", code, err)
	}

	page, err := s.players.ListPlayers(ctx, model.PlayerFilter{League: code, Limit: maxPlayersPerLeague})
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("list players for league %s: %w", code, err)
	}

	clubIndex, nameIndex := buildMatchIndexes(page.Items)

	matched := make([]model.PlayerStatistics, 0, len(rows))
	unmatched := 0
	for _, row := range rows {
		playerID, ok := clubIndex[normalizeName(row.Name)+"|"+normalizeClub(row.ClubName)]
		if !ok {
			playerID, ok = nameIndex[normalizeName(row.Name)]
		}
		if !ok {
			unmatched++
			continue
		}
		matched = append(matched, model.PlayerStatistics{
			PlayerID:      playerID,
			Goals:         row.Goals,
			Assists:       row.Assists,
			Shots:         row.Shots,
			KeyPasses:     row.KeyPasses,
			Dribbles:      row.Dribbles,
			Tackles:       row.Tackles,
			Interceptions: row.Interceptions,
			Fouls:         row.Fouls,
			Clears:        row.Clears,
			Blocks:        row.Blocks,
			WasDribbled:   row.WasDribbled,
			Rating:        row.Rating,
		})
	}

	inserted := 0
	if len(matched) > 0 {
		_, inserted, err = s.statistics.SaveStatistics(ctx, matched)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("save statistics for league %s: %w", code, err)
		}
	}

	return len(rows), len(matched), unmatched, inserted, nil
}

// buildMatchIndexes arma los dos índices de matcheo de una liga: el de
// nombre+club normalizado y el de nombre único, usado como fallback cuando
// los proveedores nombran distinto al club. Los nombres duplicados quedan
// fuera del índice de nombres para no producir matcheos falsos.
func buildMatchIndexes(players []model.Player) (map[string]int64, map[string]int64) {
	clubIndex := make(map[string]int64, len(players))
	nameCount := make(map[string]int, len(players))
	nameIndex := make(map[string]int64, len(players))

	for _, p := range players {
		nameKey := normalizeName(p.Name)
		nameCount[nameKey]++

		clubKey := nameKey + "|" + normalizeClub(p.ClubName)
		if _, exists := clubIndex[clubKey]; !exists {
			clubIndex[clubKey] = p.ID
		}
		if _, exists := nameIndex[nameKey]; !exists {
			nameIndex[nameKey] = p.ID
		}
	}

	for name, count := range nameCount {
		if count > 1 {
			delete(nameIndex, name)
		}
	}

	return clubIndex, nameIndex
}
