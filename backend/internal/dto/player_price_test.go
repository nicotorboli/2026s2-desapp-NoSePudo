package dto

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestPlayerQuoteDesdeModelo(t *testing.T) {
	quote := model.PlayerQuote{
		CreatedAt:       time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
		Strategy:        "rateada",
		StrategyVersion: "1",
		PlayerID:        42,
		Price:           model.CreditsFromUnits(51),
	}

	got := PlayerQuoteDesdeModelo(quote)

	if got.CreatedAt != "2026-11-02T10:00:00Z" {
		t.Errorf("createdAt = %q, se esperaba 2026-11-02T10:00:00Z", got.CreatedAt)
	}
	if got.Strategy != "rateada" || got.StrategyVersion != "1" {
		t.Errorf("strategy = %q/%q, se esperaba rateada/1", got.Strategy, got.StrategyVersion)
	}
	if got.Price != "51.0000" {
		t.Errorf("price = %q, se esperaba 51.0000", got.Price)
	}
	if got.PlayerID != 42 {
		t.Errorf("playerId = %d, se esperaba 42", got.PlayerID)
	}
}

func TestPlayerPriceDesdeModelo(t *testing.T) {
	score := model.PlayerScore{
		PlayerID:   42,
		ComputedAt: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
	}

	got := PlayerPriceDesdeModelo(score, 0.5, 0.4, model.CreditsFromUnits(41), "rateada")

	if got.PlayerID != 42 {
		t.Errorf("playerId = %d, se esperaba 42", got.PlayerID)
	}
	if got.ComputedAt != "2026-11-02T10:00:00Z" {
		t.Errorf("computedAt = %q, se esperaba 2026-11-02T10:00:00Z", got.ComputedAt)
	}
	if got.StrategyName != "rateada" {
		t.Errorf("strategyName = %q, se esperaba rateada", got.StrategyName)
	}
	if got.Score != 0.4 || got.LeagueCoefficient != 0.5 {
		t.Errorf("score = %v, coefficient = %v; se esperaba 0.4 y 0.5", got.Score, got.LeagueCoefficient)
	}
	if got.Price != "41.0000" {
		t.Errorf("price = %q, se esperaba 41.0000", got.Price)
	}
}
