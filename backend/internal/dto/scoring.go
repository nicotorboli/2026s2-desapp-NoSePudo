package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoringStrategyResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ActiveStrategyResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
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
