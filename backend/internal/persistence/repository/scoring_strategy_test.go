package repository_test

import (
	"database/sql"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/testutil"
)

func newStrategyRepo(t *testing.T) (*repository.ScoringStrategyRepositoryImpl, *sql.DB) {
	t.Helper()
	db := startPostgres(t)
	return repository.NewScoringStrategyRepository(dao.NewScoringStrategyDao(db)), db
}

func strategiesByID(strategies []model.ScoringStrategy) map[int]model.ScoringStrategy {
	byID := make(map[int]model.ScoringStrategy, len(strategies))
	for _, s := range strategies {
		byID[s.ID] = s
	}
	return byID
}

func TestScoringStrategyRepositoryList(t *testing.T) {
	repo, db := newStrategyRepo(t)

	testutil.InsertStrategy(t, db, model.ScoringStrategy{
		ID:      100,
		Name:    "rateada-test",
		Alpha:   0.5,
		Weights: map[string]map[string]float64{"Defender": {"tackles": 0.5, "fouls": 0.5}},
	})
	testutil.InsertStrategy(t, db, model.ScoringStrategy{
		ID:      101,
		Name:    "estadistica-test",
		Alpha:   0.2,
		Weights: map[string]map[string]float64{"Attacker": {"goals": 1.0}},
	})

	strategies, err := repo.List(t.Context())
	if err != nil {
		t.Fatalf("List devolvió error: %v", err)
	}
	byID := strategiesByID(strategies)

	got, ok := byID[100]
	if !ok {
		t.Fatal("List no devolvió la estrategia 100")
	}
	if got.Name != "rateada-test" || got.Alpha != 0.5 {
		t.Errorf("estrategia 100 = %+v", got)
	}
	if got.Weights["Defender"]["tackles"] != 0.5 || got.Weights["Defender"]["fouls"] != 0.5 {
		t.Errorf("pesos de la 100 = %+v", got.Weights)
	}

	if _, ok := byID[101]; !ok {
		t.Fatal("List no devolvió la estrategia 101")
	}
}

func TestScoringStrategyRepositoryListParsesWeights(t *testing.T) {
	repo, db := newStrategyRepo(t)

	testutil.InsertStrategy(t, db, model.ScoringStrategy{ID: 102, Name: "vacia", Alpha: 0, Weights: map[string]map[string]float64{}})
	testutil.InsertStrategy(t, db, model.ScoringStrategy{
		ID:    103,
		Name:  "anidada",
		Alpha: 1.0,
		Weights: map[string]map[string]float64{
			"Midfielder": {"key_passes": 0.6, "dribbles": 0.4},
			"Goalkeeper": {"clears": 1.0},
		},
	})

	strategies, err := repo.List(t.Context())
	if err != nil {
		t.Fatalf("List devolvió error: %v", err)
	}
	byID := strategiesByID(strategies)

	empty, ok := byID[102]
	if !ok {
		t.Fatal("List no devolvió la estrategia 102")
	}
	if len(empty.Weights) != 0 {
		t.Errorf("pesos vacíos = %+v", empty.Weights)
	}

	nested, ok := byID[103]
	if !ok {
		t.Fatal("List no devolvió la estrategia 103")
	}
	if nested.Weights["Midfielder"]["key_passes"] != 0.6 || nested.Weights["Goalkeeper"]["clears"] != 1.0 {
		t.Errorf("pesos anidados = %+v", nested.Weights)
	}
}
