package dto

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type PlayerListItemResponse struct {
	Name     string `json:"name"`
	Club     string `json:"club"`
	League   string `json:"league"`
	Position string `json:"position"`
	ID       int64  `json:"id"`
}

func PlayerListItemDesdeModelo(m model.Player) PlayerListItemResponse {
	return PlayerListItemResponse{
		ID:       m.ID,
		Name:     m.Name,
		Club:     m.ClubName,
		League:   m.LeagueName,
		Position: m.Position,
	}
}
