package model

import (
	"errors"
	"time"
)

var (
	ErrQuoteNotFound      = errors.New("el jugador no tiene una cotización vigente")
	ErrQuoteAlreadyExists = errors.New("el jugador ya tiene una cotización con esa estrategia")
)

const (
	InitialQuoteStrategy        = "initial"
	InitialQuoteStrategyVersion = "1"
)

type PlayerQuote struct {
	CreatedAt       time.Time
	ScoreID         *int64
	Strategy        string
	StrategyVersion string
	ID              int64
	PlayerID        int64
	Price           Credits
}
