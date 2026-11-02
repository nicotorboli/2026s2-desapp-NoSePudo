package testutil

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func InsertStrategy(t *testing.T, db *sql.DB, s model.ScoringStrategy) int {
	t.Helper()

	weights, err := json.Marshal(s.Weights)
	if err != nil {
		t.Fatalf("no se pudo serializar los pesos: %v", err)
	}

	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO scoring_strategies (id, name, alpha, weights) VALUES ($1, $2, $3, $4)`,
		s.ID, s.Name, s.Alpha, weights); err != nil {
		t.Fatalf("no se pudo insertar la estrategia: %v", err)
	}

	return s.ID
}

func RatingStrategy(id int) model.ScoringStrategy {
	return model.ScoringStrategy{
		ID:      id,
		Name:    "test-rating",
		Alpha:   1.0,
		Weights: map[string]map[string]float64{},
	}
}
