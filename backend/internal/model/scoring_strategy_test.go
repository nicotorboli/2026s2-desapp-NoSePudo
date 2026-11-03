package model

import (
	"math"
	"testing"
)

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestScoringStrategyScoreKnownPosition(t *testing.T) {
	s := ScoringStrategy{
		Alpha:   0.5,
		Weights: map[string]map[string]float64{"Attacker": {"goals": 1.0}},
	}

	got := s.Score("Attacker", PlayerStatistics{Rating: 8.0, Goals: 15})

	assertClose(t, got, 0.65)
}

func TestScoringStrategyScoreInvertsNegativeStats(t *testing.T) {
	s := ScoringStrategy{
		Alpha:   0,
		Weights: map[string]map[string]float64{"Defender": {"fouls": 1.0}},
	}

	assertClose(t, s.Score("Defender", PlayerStatistics{Fouls: 0}), 1.0)
	assertClose(t, s.Score("Defender", PlayerStatistics{Fouls: 3}), 0.0)
}

func TestScoringStrategyScoreFallback(t *testing.T) {
	s := ScoringStrategy{Alpha: 0, Weights: map[string]map[string]float64{}}

	// fouls y was_dribbled en 0 son crédito completo (invertidas): 2/11.
	assertClose(t, s.Score("Unknown", PlayerStatistics{}), 2.0/11.0)

	// goals al máximo suma uno más: 3/11.
	assertClose(t, s.Score("Unknown", PlayerStatistics{Goals: 30}), 3.0/11.0)
}

func TestScoringStrategyScoreKnownPositionWithoutWeightsFallsBack(t *testing.T) {
	s := ScoringStrategy{Alpha: 1.0, Weights: map[string]map[string]float64{}}

	assertClose(t, s.Score("Midfielder", PlayerStatistics{Rating: 7.0}), 0.7)
}

func TestNormalizeStat(t *testing.T) {
	cases := []struct {
		name  string
		stat  string
		stats PlayerStatistics
		want  float64
	}{
		{"goals medio", "goals", PlayerStatistics{Goals: 15}, 0.5},
		{"goals clamp", "goals", PlayerStatistics{Goals: 40}, 1.0},
		{"assists", "assists", PlayerStatistics{Assists: 10}, 0.5},
		{"shots", "shots", PlayerStatistics{Shots: 2}, 0.5},
		{"key_passes", "key_passes", PlayerStatistics{KeyPasses: 1.5}, 0.5},
		{"dribbles", "dribbles", PlayerStatistics{Dribbles: 2}, 0.5},
		{"tackles", "tackles", PlayerStatistics{Tackles: 2}, 0.5},
		{"interceptions", "interceptions", PlayerStatistics{Interceptions: 1.5}, 0.5},
		{"clears", "clears", PlayerStatistics{Clears: 3}, 0.5},
		{"blocks", "blocks", PlayerStatistics{Blocks: 1}, 0.5},
		{"fouls cero", "fouls", PlayerStatistics{Fouls: 0}, 1.0},
		{"fouls maximo", "fouls", PlayerStatistics{Fouls: 3}, 0.0},
		{"was_dribbled cero", "was_dribbled", PlayerStatistics{WasDribbled: 0}, 1.0},
		{"was_dribbled maximo", "was_dribbled", PlayerStatistics{WasDribbled: 2}, 0.0},
		{"stat desconocido", "nope", PlayerStatistics{}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := normalizeStat(c.stat, c.stats); got != c.want {
				t.Fatalf("normalizeStat(%q) = %v, want %v", c.stat, got, c.want)
			}
		})
	}
}

func TestPositionDiscount(t *testing.T) {
	for _, position := range []string{"Goalkeeper", "Defender", "Midfielder", "Attacker"} {
		if got := PositionDiscount(position); got != 1.0 {
			t.Errorf("PositionDiscount(%q) = %v, want 1.0", position, got)
		}
	}
	for _, position := range []string{"Unknown", "", "Winger"} {
		if got := PositionDiscount(position); got != UnknownPositionDiscount {
			t.Errorf("PositionDiscount(%q) = %v, want %v", position, got, UnknownPositionDiscount)
		}
	}
}
