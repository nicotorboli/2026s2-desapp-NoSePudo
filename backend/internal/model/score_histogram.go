package model

import "time"

type ScoreScope struct {
	Position   string
	LeagueCode string
}

type ScoreHistogram struct {
	ComputedAt time.Time
	Position   string
	LeagueCode string
	Buckets    []int64
	ID         int64
}
