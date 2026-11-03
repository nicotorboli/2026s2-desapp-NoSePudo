package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerPriceResponse struct {
	ComputedAt        string  `json:"computedAt"`
	StrategyName      string  `json:"strategyName"`
	Price             string  `json:"price"`
	Score             float64 `json:"score"`
	LeagueCoefficient float64 `json:"leagueCoefficient"`
	PlayerID          int64   `json:"playerId"`
}

type PlayerQuoteResponse struct {
	CreatedAt       string `json:"createdAt"`
	Strategy        string `json:"strategy"`
	StrategyVersion string `json:"strategyVersion"`
	Price           string `json:"price"`
	PlayerID        int64  `json:"playerId"`
}

func PlayerPriceDesdeModelo(score model.PlayerScore, coefficient, finalScore float64, price model.Credits, strategyName string) PlayerPriceResponse {
	return PlayerPriceResponse{
		ComputedAt:        score.ComputedAt.Format(time.RFC3339),
		StrategyName:      strategyName,
		Price:             CreditsADecimal(price),
		Score:             finalScore,
		LeagueCoefficient: coefficient,
		PlayerID:          score.PlayerID,
	}
}

func PlayerQuoteDesdeModelo(quote model.PlayerQuote) PlayerQuoteResponse {
	return PlayerQuoteResponse{
		CreatedAt:       quote.CreatedAt.Format(time.RFC3339),
		Strategy:        quote.Strategy,
		StrategyVersion: quote.StrategyVersion,
		Price:           CreditsADecimal(quote.Price),
		PlayerID:        quote.PlayerID,
	}
}
