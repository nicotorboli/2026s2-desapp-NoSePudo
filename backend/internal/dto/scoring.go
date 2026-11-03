package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoringStrategyResponse struct {
	Weights map[string]map[string]float64 `json:"weights"`
	Name    string                        `json:"name"`
	Alpha   float64                       `json:"alpha"`
	ID      int                           `json:"id"`
}

type ActiveStrategyResponse struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type LeagueCoefficientResponse struct {
	EffectiveFrom string  `json:"effectiveFrom"`
	LeagueCode    string  `json:"leagueCode"`
	Coefficient   float64 `json:"coefficient"`
}

type ActivateStrategyRequest struct {
	ID int `json:"id"`
}

type SetLeagueCoefficientRequest struct {
	Coefficient float64 `json:"coefficient"`
}

func LeagueCoefficientDesdeModelo(c model.LeagueCoefficient) LeagueCoefficientResponse {
	return LeagueCoefficientResponse{
		EffectiveFrom: c.EffectiveFrom.Format(time.RFC3339),
		LeagueCode:    c.LeagueCode,
		Coefficient:   c.Coefficient,
	}
}
