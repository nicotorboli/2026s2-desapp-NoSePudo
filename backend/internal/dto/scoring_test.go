package dto

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestLeagueCoefficientDesdeModelo(t *testing.T) {
	coefficient := model.LeagueCoefficient{
		LeagueCode:    "PL",
		Coefficient:   1.5,
		EffectiveFrom: time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
	}

	got := LeagueCoefficientDesdeModelo(coefficient)

	if got.LeagueCode != "PL" || got.Coefficient != 1.5 {
		t.Fatalf("league = %q, coefficient = %v", got.LeagueCode, got.Coefficient)
	}
	if got.EffectiveFrom == nil || *got.EffectiveFrom != "2026-11-02T10:00:00Z" {
		t.Fatalf("effectiveFrom = %v", got.EffectiveFrom)
	}
}
