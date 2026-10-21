package model

import "time"

const (
	InitialQuoteStrategy        = "initial"
	InitialQuoteStrategyVersion = "1"
)

type PlayerQuote struct {
	CreatedAt       time.Time
	Strategy        string
	StrategyVersion string
	ID              int64
	PlayerID        int64
	Price           Credits
}
