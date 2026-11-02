package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var ErrScoreNotFound = errors.New("player score not found")

const scoreMaxPlayersPerLeague = 10000

const rankingMaxPlayers = 50000

type ScoreRepository interface {
	Save(ctx context.Context, scores []model.PlayerScore) error
	HasScoreFor(ctx context.Context, playerID int64, strategyID int, statisticsID int64) (bool, error)
	History(ctx context.Context, playerID int64) ([]model.PlayerScore, error)
	LatestPercentiles(ctx context.Context, playerIDs []int64) (map[int64]float64, error)
}

type HistogramRepository interface {
	Save(ctx context.Context, h model.ScoreHistogram) (int64, error)
	Get(ctx context.Context, scope model.ScoreScope) (model.ScoreHistogram, error)
}

type StrategyActivationRepository interface {
	Activate(ctx context.Context, strategyID int, actorID int64) error
	Active(ctx context.Context) (model.StrategyActivation, bool, error)
}

type LeagueCoefficientRepository interface {
	Set(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error
	Current(ctx context.Context, leagueCode string) (model.LeagueCoefficient, bool, error)
}

type ScoreService struct {
	strategies   map[int]model.ScoringStrategy
	scores       ScoreRepository
	histograms   HistogramRepository
	activations  StrategyActivationRepository
	coefficients LeagueCoefficientRepository
	players      PlayerRepository
	statistics   PlayerStatisticsRepository
	logger       *slog.Logger
}

func NewScoreService(
	strategies map[int]model.ScoringStrategy,
	scores ScoreRepository,
	histograms HistogramRepository,
	activations StrategyActivationRepository,
	coefficients LeagueCoefficientRepository,
	players PlayerRepository,
	statistics PlayerStatisticsRepository,
	logger *slog.Logger,
) *ScoreService {
	return &ScoreService{
		strategies:   strategies,
		scores:       scores,
		histograms:   histograms,
		activations:  activations,
		coefficients: coefficients,
		players:      players,
		statistics:   statistics,
		logger:       logger,
	}
}

func (s *ScoreService) ListStrategies() []dto.ScoringStrategyResponse {
	strategies := make([]dto.ScoringStrategyResponse, 0, len(s.strategies))
	for _, strategy := range s.strategies {
		strategies = append(strategies, dto.ScoringStrategyResponse{ID: strategy.ID, Name: strategy.Name})
	}
	sort.Slice(strategies, func(i, j int) bool { return strategies[i].ID < strategies[j].ID })
	return strategies
}

func (s *ScoreService) ActiveStrategy(ctx context.Context) (dto.ActiveStrategyResponse, bool, error) {
	activation, found, err := s.activations.Active(ctx)
	if err != nil {
		return dto.ActiveStrategyResponse{}, false, fmt.Errorf("get active strategy: %w", err)
	}
	if !found {
		return dto.ActiveStrategyResponse{}, false, nil
	}
	return dto.ActiveStrategyResponse{ID: activation.StrategyID, Name: s.strategyName(activation.StrategyID)}, true, nil
}

func (s *ScoreService) ActivateStrategy(ctx context.Context, strategyID int, actorID int64) error {
	if _, ok := s.strategies[strategyID]; !ok {
		return fmt.Errorf("unknown scoring strategy %d", strategyID)
	}

	if err := s.activations.Activate(ctx, strategyID, actorID); err != nil {
		return err
	}

	page, err := s.players.ListPlayers(ctx, model.PlayerFilter{Limit: scoreMaxPlayersPerLeague})
	if err != nil {
		return fmt.Errorf("list players: %w", err)
	}

	return s.recalculate(ctx, strategyID, page.Items)
}

func (s *ScoreService) SetLeagueCoefficient(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error {
	if coefficient <= 0 {
		return fmt.Errorf("league coefficient must be positive")
	}
	return s.coefficients.Set(ctx, leagueCode, coefficient, actorID)
}

func (s *ScoreService) GetLeagueCoefficient(ctx context.Context, leagueCode string) (dto.LeagueCoefficientResponse, bool, error) {
	coefficient, found, err := s.coefficients.Current(ctx, leagueCode)
	if err != nil {
		return dto.LeagueCoefficientResponse{}, false, fmt.Errorf("get league coefficient: %w", err)
	}
	if !found {
		return dto.LeagueCoefficientResponse{}, false, nil
	}
	return dto.LeagueCoefficientDesdeModelo(coefficient), true, nil
}

func (s *ScoreService) RecalculateAfterSync(ctx context.Context, leagueCode string) error {
	activation, found, err := s.activations.Active(ctx)
	if err != nil {
		return fmt.Errorf("get active strategy: %w", err)
	}
	if !found {
		return nil
	}

	page, err := s.players.ListPlayers(ctx, model.PlayerFilter{League: leagueCode, Limit: scoreMaxPlayersPerLeague})
	if err != nil {
		return fmt.Errorf("list players for league %s: %w", leagueCode, err)
	}

	return s.recalculate(ctx, activation.StrategyID, page.Items)
}

func (s *ScoreService) GetScore(ctx context.Context, playerID int64) (dto.PlayerScoreResponse, error) {
	player, err := s.players.GetPlayerByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.PlayerScoreResponse{}, ErrNotFound
		}
		return dto.PlayerScoreResponse{}, err
	}

	history, err := s.scores.History(ctx, playerID)
	if err != nil {
		return dto.PlayerScoreResponse{}, fmt.Errorf("get player score history: %w", err)
	}
	if len(history) == 0 {
		return dto.PlayerScoreResponse{}, ErrScoreNotFound
	}

	coefficient := s.leagueCoefficient(ctx, player.LeagueCode)
	resp := dto.PlayerScoreDesdeModelo(history[0], coefficient, s.strategyName(history[0].StrategyID))
	resp.Score *= model.PositionDiscount(player.Position)
	return resp, nil
}

func (s *ScoreService) GetScoreHistory(ctx context.Context, playerID int64) ([]dto.PlayerScoreResponse, error) {
	player, err := s.players.GetPlayerByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	history, err := s.scores.History(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get player score history: %w", err)
	}

	coefficient := s.leagueCoefficient(ctx, player.LeagueCode)
	discount := model.PositionDiscount(player.Position)
	out := make([]dto.PlayerScoreResponse, len(history))
	for i, score := range history {
		resp := dto.PlayerScoreDesdeModelo(score, coefficient, s.strategyName(score.StrategyID))
		resp.Score *= discount
		out[i] = resp
	}

	return out, nil
}

func (s *ScoreService) leagueCoefficient(ctx context.Context, leagueCode string) float64 {
	coefficient, found, err := s.coefficients.Current(ctx, leagueCode)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to load league coefficient", "league_code", leagueCode, "error", err.Error())
		return 1.0
	}
	if !found {
		return 1.0
	}
	return coefficient.Coefficient
}

func (s *ScoreService) strategyName(strategyID int) string {
	if strategy, ok := s.strategies[strategyID]; ok {
		return strategy.Name
	}
	return ""
}

func (s *ScoreService) recalculate(ctx context.Context, strategyID int, players []model.Player) error {
	strategy, ok := s.strategies[strategyID]
	if !ok {
		return fmt.Errorf("unknown scoring strategy %d", strategyID)
	}

	type candidate struct {
		player model.Player
		stats  model.PlayerStatistics
		raw    float64
	}

	candidates := make([]candidate, 0, len(players))
	rawsByScope := make(map[model.ScoreScope][]float64)

	for _, p := range players {
		statsList, err := s.statistics.GetPlayerStatistics(ctx, p.ID)
		if err != nil {
			return fmt.Errorf("load statistics for player %d: %w", p.ID, err)
		}
		if len(statsList) == 0 {
			continue
		}

		latest := statsList[0]
		raw := strategy.Score(p.Position, latest)
		candidates = append(candidates, candidate{player: p, stats: latest, raw: raw})

		scope := model.ScoreScope{Position: p.Position, LeagueCode: p.LeagueCode}
		rawsByScope[scope] = append(rawsByScope[scope], raw)
	}

	bucketsByScope := make(map[model.ScoreScope][]int64, len(rawsByScope))
	for scope, raws := range rawsByScope {
		bucketsByScope[scope] = model.BuildHistogram(raws, model.ScoreHistogramBuckets)
	}

	weights := strategy.Weights
	histIDByScope := make(map[model.ScoreScope]int64)
	toSave := make([]model.PlayerScore, 0, len(candidates))

	for _, c := range candidates {
		exists, err := s.scores.HasScoreFor(ctx, c.player.ID, strategyID, c.stats.ID)
		if err != nil {
			return fmt.Errorf("check existing score for player %d: %w", c.player.ID, err)
		}
		if exists {
			continue
		}

		scope := model.ScoreScope{Position: c.player.Position, LeagueCode: c.player.LeagueCode}
		histogramID, saved := histIDByScope[scope]
		if !saved {
			histogramID, err = s.histograms.Save(ctx, model.ScoreHistogram{
				Position:   scope.Position,
				LeagueCode: scope.LeagueCode,
				Buckets:    bucketsByScope[scope],
			})
			if err != nil {
				return fmt.Errorf("save histogram for %s/%s: %w", scope.Position, scope.LeagueCode, err)
			}
			histIDByScope[scope] = histogramID
		}

		toSave = append(toSave, model.PlayerScore{
			StrategyID:   strategyID,
			Weights:      weights,
			RawScore:     c.raw,
			Percentile:   model.Percentile(bucketsByScope[scope], c.raw),
			PlayerID:     c.player.ID,
			StatisticsID: c.stats.ID,
			HistogramID:  histogramID,
		})
	}

	if err := s.scores.Save(ctx, toSave); err != nil {
		return fmt.Errorf("save player scores: %w", err)
	}

	s.logger.InfoContext(ctx, "Player scores recalculated", "strategy_id", strategyID, "computed", len(toSave))

	return nil
}

func (s *ScoreService) ListRanking(ctx context.Context, filter dto.PlayerFilterDTO, sortOrder string, offset int) (dto.PlayerRankingPageResponse, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	switch sortOrder {
	case "name":
		return s.listRankingByName(ctx, filter, offset)
	case "", "score_desc", "score_asc":
		return s.listRankingByScore(ctx, filter, offset, sortOrder == "score_asc")
	default:
		return dto.PlayerRankingPageResponse{}, fmt.Errorf("invalid ranking sort %q", sortOrder)
	}
}

func (s *ScoreService) listRankingByName(ctx context.Context, filter dto.PlayerFilterDTO, offset int) (dto.PlayerRankingPageResponse, error) {
	modelFilter := dto.PlayerFilterAModelo(filter)
	modelFilter.Page = offset/filter.Limit + 1

	page, err := s.players.ListPlayers(ctx, modelFilter)
	if err != nil {
		return dto.PlayerRankingPageResponse{}, fmt.Errorf("list players for ranking: %w", err)
	}

	items, err := s.rankingItems(ctx, page.Items)
	if err != nil {
		return dto.PlayerRankingPageResponse{}, err
	}

	return dto.PlayerRankingPageResponse{
		Items:  items,
		Offset: offset,
		Limit:  filter.Limit,
		Total:  int(page.Total),
	}, nil
}

func (s *ScoreService) listRankingByScore(ctx context.Context, filter dto.PlayerFilterDTO, offset int, ascending bool) (dto.PlayerRankingPageResponse, error) {
	modelFilter := dto.PlayerFilterAModelo(filter)
	modelFilter.Page = 1
	modelFilter.Limit = rankingMaxPlayers

	page, err := s.players.ListPlayers(ctx, modelFilter)
	if err != nil {
		return dto.PlayerRankingPageResponse{}, fmt.Errorf("list players for ranking: %w", err)
	}

	items, err := s.rankingItems(ctx, page.Items)
	if err != nil {
		return dto.PlayerRankingPageResponse{}, err
	}

	sortRanking(items, ascending)

	total := len(items)
	start := offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}

	pageItems := []dto.PlayerRankingItemResponse{}
	if start < total {
		pageItems = items[start:end]
	}

	return dto.PlayerRankingPageResponse{
		Items:  pageItems,
		Offset: offset,
		Limit:  filter.Limit,
		Total:  total,
	}, nil
}

func (s *ScoreService) rankingItems(ctx context.Context, players []model.Player) ([]dto.PlayerRankingItemResponse, error) {
	ids := make([]int64, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}

	percentiles, err := s.scores.LatestPercentiles(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load latest percentiles: %w", err)
	}

	coefficientCache := make(map[string]float64)
	items := make([]dto.PlayerRankingItemResponse, 0, len(players))
	for _, p := range players {
		item := dto.PlayerRankingItemResponse{
			ID:       p.ID,
			Name:     p.Name,
			Club:     p.ClubName,
			League:   p.LeagueName,
			Position: p.Position,
		}
		if percentile, ok := percentiles[p.ID]; ok {
			coefficient, cached := coefficientCache[p.LeagueCode]
			if !cached {
				coefficient = s.leagueCoefficient(ctx, p.LeagueCode)
				coefficientCache[p.LeagueCode] = coefficient
			}
			score := percentile * coefficient * model.PositionDiscount(p.Position)
			item.Score = &score
		}
		items = append(items, item)
	}

	return items, nil
}

func sortRanking(items []dto.PlayerRankingItemResponse, ascending bool) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == nil && items[j].Score == nil {
			return false
		}
		if items[i].Score == nil {
			return false
		}
		if items[j].Score == nil {
			return true
		}
		if ascending {
			return *items[i].Score < *items[j].Score
		}
		return *items[i].Score > *items[j].Score
	})
}
