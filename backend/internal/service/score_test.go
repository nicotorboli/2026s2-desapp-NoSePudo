package service_test

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/testutil"
)

const testStrategyID = 100

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

	testutil.InsertStrategy(t, db, testutil.RatingStrategy(testStrategyID))

	strategies, err := repos.ScoringStrategy.List(t.Context())
	if err != nil {
		t.Fatalf("no se pudieron cargar las estrategias: %v", err)
	}
	registry := make(map[int]model.ScoringStrategy, len(strategies))
	for _, strategy := range strategies {
		registry[strategy.ID] = strategy
	}

	svc := service.NewScoreService(
		registry,
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

func (f scoreServiceFixture) seedRatedPlayer(t *testing.T, name string, rating float64) int64 {
	t.Helper()
	return f.seedRatedPlayerAt(t, name, "Midfielder", rating)
}

func (f scoreServiceFixture) seedRatedPlayerAt(t *testing.T, name, position string, rating float64) int64 {
	t.Helper()
	serviceSeedSeq++

	var playerID int64
	if err := f.db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES ($1, $2, 'Arsenal', 'Premier League', 'PL', $3) RETURNING id`,
		serviceSeedSeq, name, position).Scan(&playerID); err != nil {
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
	high := f.seedRatedPlayer(t, "Alto", 8.0)
	low := f.seedRatedPlayer(t, "Bajo", 6.0)

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
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

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}
	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
		t.Fatalf("la segunda ActivateStrategy devolvió error: %v", err)
	}

	var count int
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM player_scores WHERE strategy_id = $1`, testStrategyID).Scan(&count); err != nil {
		t.Fatalf("no se pudo contar los scores: %v", err)
	}
	if count != 1 {
		t.Errorf("scores de la estrategia %d = %d, want 1", testStrategyID, count)
	}
}

func TestScoreServiceApplyLeagueCoefficientOnRead(t *testing.T) {
	f := newScoreServiceFixture(t)
	playerID := f.seedRatedPlayer(t, "Uno", 8.0)

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
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

func TestScoreServiceAppliesUnknownPositionDiscount(t *testing.T) {
	f := newScoreServiceFixture(t)
	playerID := f.seedRatedPlayerAt(t, "Desconocido", "Unknown", 8.0)

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}

	score, err := f.svc.GetScore(t.Context(), playerID)
	if err != nil {
		t.Fatalf("GetScore devolvió error: %v", err)
	}

	want := score.Percentile * score.LeagueCoefficient * model.UnknownPositionDiscount
	if score.Score != want {
		t.Errorf("score = %v, want %v (percentil %v, coef %v, descuento %v)",
			score.Score, want, score.Percentile, score.LeagueCoefficient, model.UnknownPositionDiscount)
	}
}

func TestScoreServiceActivateUnknownStrategy(t *testing.T) {
	f := newScoreServiceFixture(t)

	if err := f.svc.ActivateStrategy(t.Context(), 999, f.actorID); err == nil {
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
	playerID := f.seedRatedPlayer(t, "SinScore", 7.0)

	if _, err := f.svc.GetScore(t.Context(), playerID); !errors.Is(err, service.ErrScoreNotFound) {
		t.Fatalf("GetScore devolvió %v, want ErrScoreNotFound", err)
	}
}

func TestScoreServiceListRankingOrdersByScore(t *testing.T) {
	f := newScoreServiceFixture(t)
	f.seedRatedPlayer(t, "Bajo", 6.0)
	f.seedRatedPlayer(t, "Alto", 8.0)

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}

	ranking, err := f.svc.ListRanking(t.Context(), dto.PlayerFilterDTO{Limit: 25}, "score_desc", 0)
	if err != nil {
		t.Fatalf("ListRanking devolvió error: %v", err)
	}
	if len(ranking.Items) != 2 {
		t.Fatalf("items = %d, se esperaban 2", len(ranking.Items))
	}
	if ranking.Items[0].Name != "Alto" || ranking.Items[1].Name != "Bajo" {
		t.Errorf("orden = %q, %q; se esperaba Alto, Bajo", ranking.Items[0].Name, ranking.Items[1].Name)
	}
	if ranking.Items[0].Score == nil || ranking.Items[1].Score == nil {
		t.Fatal("se esperaban scores no nulos")
	}
	if *ranking.Items[0].Score <= *ranking.Items[1].Score {
		t.Errorf("scores desordenados: %v, %v", *ranking.Items[0].Score, *ranking.Items[1].Score)
	}
}

func TestScoreServiceListRankingSortsMissingScoresLast(t *testing.T) {
	f := newScoreServiceFixture(t)
	f.seedRatedPlayer(t, "ConScore", 8.0)

	if err := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); err != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", err)
	}

	f.seedRatedPlayer(t, "SinScore", 9.0)

	ranking, err := f.svc.ListRanking(t.Context(), dto.PlayerFilterDTO{Limit: 25}, "score_desc", 0)
	if err != nil {
		t.Fatalf("ListRanking devolvió error: %v", err)
	}
	if len(ranking.Items) != 2 {
		t.Fatalf("items = %d, se esperaban 2", len(ranking.Items))
	}
	if ranking.Items[0].Name != "ConScore" || ranking.Items[1].Name != "SinScore" {
		t.Errorf("orden = %q, %q; se esperaba ConScore, SinScore", ranking.Items[0].Name, ranking.Items[1].Name)
	}
	if ranking.Items[1].Score != nil {
		t.Errorf("el jugador sin score debería quedar al final con nil")
	}
}

func TestScoreServiceEnsureDefaultActivationIsIdempotentAndRespectsOperatorChoice(t *testing.T) {
	f := newScoreServiceFixture(t)

	if err := f.svc.EnsureDefaultActivation(t.Context(), f.actorID); err != nil {
		t.Fatalf("EnsureDefaultActivation devolvió error: %v", err)
	}

	active, found, err := f.svc.ActiveStrategy(t.Context())
	if err != nil {
		t.Fatalf("ActiveStrategy devolvió error: %v", err)
	}
	if !found || active.ID != 1 {
		t.Fatalf("la estrategia activa por defecto debería ser 1, obtuve %+v (found=%v)", active, found)
	}

	if ensureErr := f.svc.EnsureDefaultActivation(t.Context(), f.actorID); ensureErr != nil {
		t.Fatalf("el segundo EnsureDefaultActivation devolvió error: %v", ensureErr)
	}

	active, _, err = f.svc.ActiveStrategy(t.Context())
	if err != nil {
		t.Fatalf("ActiveStrategy después del segundo ensure: %v", err)
	}
	if active.ID != 1 {
		t.Errorf("el segundo ensure cambió la estrategia activa a %d", active.ID)
	}

	if activateErr := f.svc.ActivateStrategy(t.Context(), testStrategyID, f.actorID); activateErr != nil {
		t.Fatalf("ActivateStrategy devolvió error: %v", activateErr)
	}
	if ensureErr := f.svc.EnsureDefaultActivation(t.Context(), f.actorID); ensureErr != nil {
		t.Fatalf("EnsureDefaultActivation después de una activación manual: %v", ensureErr)
	}

	active, _, err = f.svc.ActiveStrategy(t.Context())
	if err != nil {
		t.Fatalf("ActiveStrategy final: %v", err)
	}
	if active.ID != testStrategyID {
		t.Errorf("EnsureDefaultActivation pisó la elección del operador: %d", active.ID)
	}
}
