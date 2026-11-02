package dto

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestPlayerScoreDesdeModelo(t *testing.T) {
	score := model.PlayerScore{
		StrategyID: 1,
		Percentile: 0.8,
		RawScore:   0.7,
		PlayerID:   42,
		ComputedAt: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
	}

	got := PlayerScoreDesdeModelo(score, 0.5, "standard")

	if got.Score != 0.4 {
		t.Fatalf("score = %v, want 0.4", got.Score)
	}
	if got.StrategyID != 1 || got.StrategyName != "standard" {
		t.Fatalf("strategy = %d/%q, want 1/\"standard\"", got.StrategyID, got.StrategyName)
	}
}
