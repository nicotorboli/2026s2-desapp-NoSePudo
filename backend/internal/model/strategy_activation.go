package model

import "time"

type StrategyActivation struct {
	ActivatedAt time.Time
	StrategyID  int
	ActivatedBy int64
	ID          int64
}
