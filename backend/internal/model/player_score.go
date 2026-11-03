package model

import "time"

type PlayerScore struct {
	ComputedAt   time.Time
	Weights      map[string]map[string]float64
	StrategyID   int
	RawScore     float64
	Percentile   float64
	PlayerID     int64
	StatisticsID int64
	HistogramID  int64
	ID           int64
}
