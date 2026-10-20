package whoscored

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type statisticsFeed struct {
	PlayerTableStats []playerTableStat `json:"playerTableStats"`
	Paging           paging            `json:"paging"`
}

type paging struct {
	Page        int `json:"page"`
	NumberPages int `json:"numberPages"`
}

// playerTableStat es la fila plana que devuelve el feed: cada categoría trae
// sus propias columnas y los campos ausentes quedan en cero.
type playerTableStat struct {
	Name                   string  `json:"name"`
	TeamName               string  `json:"teamName"`
	Goal                   float64 `json:"goal"`
	AssistTotal            float64 `json:"assistTotal"`
	ShotsPerGame           float64 `json:"shotsPerGame"`
	KeyPassPerGame         float64 `json:"keyPassPerGame"`
	DribbleWonPerGame      float64 `json:"dribbleWonPerGame"`
	TacklePerGame          float64 `json:"tacklePerGame"`
	InterceptionPerGame    float64 `json:"interceptionPerGame"`
	FoulsPerGame           float64 `json:"foulsPerGame"`
	ClearancePerGame       float64 `json:"clearancePerGame"`
	OutfielderBlockPerGame float64 `json:"outfielderBlockPerGame"`
	WasDribbledPerGame     float64 `json:"wasDribbledPerGame"`
	PlayerID               int64   `json:"playerId"`
	Rating                 float64 `json:"rating"`
}

func parseFeedPage(body []byte) ([]playerTableStat, error) {
	var feed statisticsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("unmarshal statistics feed: %w", err)
	}
	return feed.PlayerTableStats, nil
}

// mergeFeeds combina el feed ofensivo con el defensivo por playerId: del
// ofensivo salen el nombre, el club y las métricas de ataque; del defensivo,
// las entradas, intercepciones, faltas, despejes, bloqueos y gambetas recibidas.
func mergeFeeds(offensive, defensive []playerTableStat) []model.PlayerStatisticsRow {
	defensiveByPlayer := make(map[int64]playerTableStat, len(defensive))
	for _, row := range defensive {
		defensiveByPlayer[row.PlayerID] = row
	}

	rows := make([]model.PlayerStatisticsRow, 0, len(offensive))
	for _, row := range offensive {
		def := defensiveByPlayer[row.PlayerID]
		rows = append(rows, model.PlayerStatisticsRow{
			Name:          strings.TrimSpace(row.Name),
			ClubName:      strings.TrimSpace(row.TeamName),
			Goals:         int(row.Goal + 0.5),
			Assists:       int(row.AssistTotal + 0.5),
			Shots:         row.ShotsPerGame,
			KeyPasses:     row.KeyPassPerGame,
			Dribbles:      row.DribbleWonPerGame,
			Tackles:       def.TacklePerGame,
			Interceptions: def.InterceptionPerGame,
			Fouls:         def.FoulsPerGame,
			Clears:        def.ClearancePerGame,
			Blocks:        def.OutfielderBlockPerGame,
			WasDribbled:   def.WasDribbledPerGame,
			Rating:        row.Rating,
		})
	}

	return rows
}
