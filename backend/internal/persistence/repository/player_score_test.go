package repository_test

import (
	"database/sql"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type scoreFixture struct {
	repo         *repository.PlayerScoreRepositoryImpl
	db           *sql.DB
	playerID     int64
	statisticsID int64
	histogramID  int64
}

func newScoreFixture(t *testing.T) scoreFixture {
	t.Helper()
	ctx := t.Context()
	db := startPostgres(t)

	var playerID int64
	if err := db.QueryRowContext(ctx,
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES (1, 'Ernesto', 'Arsenal', 'Premier League', 'PL', 'Midfielder') RETURNING id`).Scan(&playerID); err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}

	var statisticsID int64
	if err := db.QueryRowContext(ctx,
		`INSERT INTO player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, interceptions, fouls, clears, blocks, was_dribbled, rating)
		 VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7.0) RETURNING id`, playerID).Scan(&statisticsID); err != nil {
		t.Fatalf("no se pudo sembrar estadísticas: %v", err)
	}

	histogramID, err := repository.NewScoreHistogramRepository(dao.NewScoreHistogramDao(db)).
		Save(ctx, model.ScoreHistogram{Position: "Midfielder", LeagueCode: "PL", Buckets: []int64{0, 1}})
	if err != nil {
		t.Fatalf("no se pudo sembrar un histograma: %v", err)
	}

	return scoreFixture{
		repo:         repository.NewPlayerScoreRepository(db, dao.NewPlayerScoreDao(db)),
		db:           db,
		playerID:     playerID,
		statisticsID: statisticsID,
		histogramID:  histogramID,
	}
}

func (f scoreFixture) aScore() model.PlayerScore {
	return model.PlayerScore{
		StrategyID:   1,
		Weights:      map[string]map[string]float64{"Midfielder": {"rating": 1.0}},
		RawScore:     0.7,
		Percentile:   0.8,
		PlayerID:     f.playerID,
		StatisticsID: f.statisticsID,
		HistogramID:  f.histogramID,
	}
}

func TestPlayerScoreRepositorySaveAndReadBack(t *testing.T) {
	f := newScoreFixture(t)

	if err := f.repo.Save(t.Context(), []model.PlayerScore{f.aScore()}); err != nil {
		t.Fatalf("Save devolvió error: %v", err)
	}

	history, err := f.repo.History(t.Context(), f.playerID)
	if err != nil {
		t.Fatalf("History devolvió error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("History devolvió %d scores, se esperaba 1", len(history))
	}
	if history[0].StrategyID != 1 || history[0].Percentile != 0.8 {
		t.Errorf("score = %+v", history[0])
	}
}

func TestPlayerScoreRepositoryHasScoreFor(t *testing.T) {
	f := newScoreFixture(t)
	ctx := t.Context()

	found, err := f.repo.HasScoreFor(ctx, f.playerID, 1, f.statisticsID)
	if err != nil {
		t.Fatalf("HasScoreFor devolvió error: %v", err)
	}
	if found {
		t.Error("HasScoreFor = true sin haber guardado")
	}

	if err = f.repo.Save(ctx, []model.PlayerScore{f.aScore()}); err != nil {
		t.Fatalf("Save devolvió error: %v", err)
	}

	found, err = f.repo.HasScoreFor(ctx, f.playerID, 1, f.statisticsID)
	if err != nil {
		t.Fatalf("HasScoreFor devolvió error: %v", err)
	}
	if !found {
		t.Error("HasScoreFor = false después de guardar")
	}
}

func TestPlayerScoreRepositoryRejectsDuplicate(t *testing.T) {
	f := newScoreFixture(t)
	ctx := t.Context()

	if err := f.repo.Save(ctx, []model.PlayerScore{f.aScore()}); err != nil {
		t.Fatalf("la primera Save devolvió error: %v", err)
	}
	if err := f.repo.Save(ctx, []model.PlayerScore{f.aScore()}); err == nil {
		t.Fatal("la segunda Save no rechazó el duplicado (UNIQUE)")
	}
}

func TestPlayerScoreRepositoryEmptyHistory(t *testing.T) {
	f := newScoreFixture(t)

	history, err := f.repo.History(t.Context(), f.playerID)
	if err != nil {
		t.Fatalf("History devolvió error: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("History devolvió %d scores, se esperaba 0", len(history))
	}
}

func TestPlayerScoreRepositoryLatestPercentiles(t *testing.T) {
	f := newScoreFixture(t)
	ctx := t.Context()

	if err := f.repo.Save(ctx, []model.PlayerScore{f.aScore()}); err != nil {
		t.Fatalf("Save devolvió error: %v", err)
	}

	percentiles, err := f.repo.LatestPercentiles(ctx, []int64{f.playerID})
	if err != nil {
		t.Fatalf("LatestPercentiles devolvió error: %v", err)
	}
	if got, ok := percentiles[f.playerID]; !ok || got != 0.8 {
		t.Errorf("percentile = %v (ok=%v), se esperaba 0.8", got, ok)
	}

	empty, err := f.repo.LatestPercentiles(ctx, nil)
	if err != nil {
		t.Fatalf("LatestPercentiles(nil) devolvió error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("LatestPercentiles(nil) = %v, se esperaba vacío", empty)
	}
}
