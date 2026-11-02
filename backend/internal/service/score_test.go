package service_test

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

// testScoringStrategy es un fixture (no una estrategia de producción): score
// determinista rating/10 para poder predecir percentiles.
type testScoringStrategy struct{ id int }

func (s testScoringStrategy) ID() int      { return s.id }
func (s testScoringStrategy) Name() string { return "test" }
func (s testScoringStrategy) Weights() map[string]map[string]float64 {
	return map[string]map[string]float64{"Midfielder": {"rating": 1.0}}
}
func (s testScoringStrategy) Score(position string, stats model.PlayerStatistics) float64 {
	return stats.Rating / 10.0
}

type scoreServiceFixture struct {
	svc     *service.ScoreService
	db      *sql.DB
	actorID int64
}

func newScoreServiceFixture(t *testing.T) scoreServiceFixture {
	t.Helper()
	db := startStatisticsPostgres(t)
	daos := dao.NewContainer(db)
	repos := repository.NewContainer(db, daos)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	user, err := repos.User.Insert(t.Context(), model.User{
		Email:        "admin@nosepudo.ar",
		PasswordHash: "x",
		Privilege:    model.PrivilegeSuperuser,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	svc := service.NewScoreService(
		map[int]service.ScoringStrategy{1: testScoringStrategy{id: 1}, 2: testScoringStrategy{id: 2}},
		repos.PlayerScore,
		repos.ScoreHistogram,
		repos.StrategyActivation,
		repos.LeagueCoefficient,
		repos.Player,
		repos.PlayerStatistics,
		logger,
	)

	return scoreServiceFixture{svc: svc, db: db, actorID: user.ID}
}

// seedRatedPlayer inserta un jugador (Midfielder, PL) con una estadística de
// rating dada y devuelve su id.
func (f scoreServiceFixture) seedRatedPlayer(t *testing.T, name string, rating float64) int64 {
	t.Helper()
	serviceSeedSeq++

	var playerID int64
	if err := f.db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES ($1, $2, 'Arsenal', 'Premier League', 'PL', 'Midfielder') RETURNING id`,
		serviceSeedSeq, name).Scan(&playerID); err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}
	if _, err := f.db.ExecContext(t.Context(),
		`INSERT INTO player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, interceptions, fouls, clears, blocks, was_dribbled, rating)
		 VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, $2)`,
		playerID, rating); err != nil {
		t.Fatalf("no se pudo sembrar estadísticas: %v", err)
	}

	return playerID
}

func TestScoreServiceRecalculatePersistsPercentile(t *testing.T) {
	f := newScoreServiceFixture(t)
	high := f.seedRatedPlayer(t, "Alto", 8.0) // raw 0.8
	low := f.seedRatedPlayer(t, "Bajo", 6.0)  // raw 0.6

	if err := f.svc.ActivateStrategy(t.Context(), 1, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}

	highScore, err := f.svc.GetScore(t.Context(), high)
	if err != nil {
		t.Fatalf("GetScore devolvió error: %v", err)
	}
	if highScore.Percentile != 0.75 {
		t.Errorf("percentil del alto = %v, want 0.75", highScore.Percentile)
	}

	lowScore, err := f.svc.GetScore(t.Context(), low)
	if err != nil {
		t.Fatalf("GetScore devolvió error: %v", err)
	}
	if lowScore.Percentile != 0.25 {
		t.Errorf("percentil del bajo = %v, want 0.25", lowScore.Percentile)
	}
}

func TestScoreServiceRecalculateIsIdempotent(t *testing.T) {
	f := newScoreServiceFixture(t)
	f.seedRatedPlayer(t, "Uno", 7.0)

	if err := f.svc.ActivateStrategy(t.Context(), 1, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}
	if err := f.svc.ActivateStrategy(t.Context(), 1, f.actorID); err != nil {
		t.Fatalf("la segunda ActivateStrategy devolvió error: %v", err)
	}

	var count int
	if err := f.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM player_scores WHERE strategy_id = 1`).Scan(&count); err != nil {
		t.Fatalf("no se pudo contar los scores: %v", err)
	}
	if count != 1 {
		t.Errorf("scores de la estrategia 1 = %d, want 1", count)
	}
}

func TestScoreServiceApplyLeagueCoefficientOnRead(t *testing.T) {
	f := newScoreServiceFixture(t)
	playerID := f.seedRatedPlayer(t, "Uno", 8.0)

	if err := f.svc.ActivateStrategy(t.Context(), 1, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}
	if err := f.svc.SetLeagueCoefficient(t.Context(), "PL", 0.5, f.actorID); err != nil {
		t.Fatalf("SetLeagueCoefficient devolvió error: %v", err)
	}

	score, err := f.svc.GetScore(t.Context(), playerID)
	if err != nil {
		t.Fatalf("GetScore devolvió error: %v", err)
	}
	if score.LeagueCoefficient != 0.5 || score.Score != score.Percentile*0.5 {
		t.Errorf("coef = %v, score = %v (percentile = %v)", score.LeagueCoefficient, score.Score, score.Percentile)
	}
}

func TestScoreServiceActivateUnknownStrategy(t *testing.T) {
	f := newScoreServiceFixture(t)

	if err := f.svc.ActivateStrategy(t.Context(), 99, f.actorID); err == nil {
		t.Fatal("ActivateStrategy con id desconocido no devolvió error")
	}
}

func TestScoreServiceRejectsNonPositiveCoefficient(t *testing.T) {
	f := newScoreServiceFixture(t)

	if err := f.svc.SetLeagueCoefficient(t.Context(), "PL", 0, f.actorID); err == nil {
		t.Fatal("SetLeagueCoefficient con 0 no devolvió error")
	}
}

func TestScoreServiceGetScoreForPlayerWithoutScore(t *testing.T) {
	f := newScoreServiceFixture(t)
	playerID := f.seedRatedPlayer(t, "SinScore", 7.0) // tiene stats pero no se activó estrategia

	if _, err := f.svc.GetScore(t.Context(), playerID); !errors.Is(err, service.ErrScoreNotFound) {
		t.Fatalf("GetScore devolvió %v, want ErrScoreNotFound", err)
	}
}
