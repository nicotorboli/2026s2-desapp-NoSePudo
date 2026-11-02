package model

import "time"

type ScoreScope struct {
	Position   string
	LeagueCode string
}

type ScoreHistogram struct {
	ComputedAt time.Time
	Buckets    []int64
	Position   string
	LeagueCode string
	ID         int64
}
