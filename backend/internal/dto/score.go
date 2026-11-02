package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerScoreResponse struct {
	ComputedAt        string  `json:"computedAt"`
	StrategyID        int     `json:"strategyId"`
	StrategyName      string  `json:"strategyName"`
	RawScore          float64 `json:"rawScore"`
	Percentile        float64 `json:"percentile"`
	Score             float64 `json:"score"`
	LeagueCoefficient float64 `json:"leagueCoefficient"`
	PlayerID          int64   `json:"playerId"`
}

func PlayerScoreDesdeModelo(s model.PlayerScore, coefficient float64, strategyName string) PlayerScoreResponse {
	return PlayerScoreResponse{
		ComputedAt:        s.ComputedAt.Format(time.RFC3339),
		StrategyID:        s.StrategyID,
		StrategyName:      strategyName,
		RawScore:          s.RawScore,
		Percentile:        s.Percentile,
		Score:             s.Percentile * coefficient,
		LeagueCoefficient: coefficient,
		PlayerID:          s.PlayerID,
	}
}
