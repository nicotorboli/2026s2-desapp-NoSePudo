package model

import "time"

type LeagueCoefficient struct {
	EffectiveFrom time.Time
	LeagueCode    string
	Coefficient   float64
	SetBy         int64
	ID            int64
}
