package service

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type ScoringStrategy interface {
	ID() int
	Name() string
	Weights() map[string]map[string]float64
	Score(position string, stats model.PlayerStatistics) float64
}
