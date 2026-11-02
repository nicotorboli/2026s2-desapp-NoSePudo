package model

import (
	"time"
)

type PlayerStatistics struct {
	CreatedAt     time.Time
	Rating        float64
	PlayerID      int64
	ID            int64
	Goals         int
	Assists       int
	Shots         float64
	KeyPasses     float64
	Dribbles      float64
	Tackles       float64
	Interceptions float64
	Fouls         float64
	Clears        float64
	Blocks        float64
	WasDribbled   float64
}
