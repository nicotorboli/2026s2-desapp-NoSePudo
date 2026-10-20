package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerStatisticsResponse struct {
	CreatedAt     string  `json:"createdAt"`
	Goals         int     `json:"goals"`
	Assists       int     `json:"assists"`
	Shots         float64 `json:"shots"`
	KeyPasses     float64 `json:"keyPasses"`
	Dribbles      float64 `json:"dribbles"`
	Tackles       float64 `json:"tackles"`
	Interceptions float64 `json:"interceptions"`
	Fouls         float64 `json:"fouls"`
	Clears        float64 `json:"clears"`
	Blocks        float64 `json:"blocks"`
	WasDribbled   float64 `json:"wasDribbled"`
	Rating        float64 `json:"rating"`
}

func PlayerStatisticsDesdeModelo(m model.PlayerStatistics) PlayerStatisticsResponse {
	return PlayerStatisticsResponse{
		Goals:         m.Goals,
		Assists:       m.Assists,
		Shots:         m.Shots,
		KeyPasses:     m.KeyPasses,
		Dribbles:      m.Dribbles,
		Tackles:       m.Tackles,
		Interceptions: m.Interceptions,
		Fouls:         m.Fouls,
		Clears:        m.Clears,
		Blocks:        m.Blocks,
		WasDribbled:   m.WasDribbled,
		Rating:        m.Rating,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
	}
}
