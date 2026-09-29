package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerDetailResponse struct {
	DateOfBirth *string `json:"dateOfBirth"`
	Nationality *string `json:"nationality"`
	ShirtNumber *int    `json:"shirtNumber"`
	Name        string  `json:"name"`
	Club        string  `json:"club"`
	League      string  `json:"league"`
	LeagueCode  string  `json:"leagueCode"`
	Position    string  `json:"position"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	ID          int64   `json:"id"`
	ExternalID  int64   `json:"externalId"`
	Active      bool    `json:"active"`
}

func PlayerDetailDesdeModelo(m model.Player) PlayerDetailResponse {
	return PlayerDetailResponse{
		ID:          m.ID,
		ExternalID:  m.ExternalID,
		Name:        m.Name,
		Club:        m.ClubName,
		League:      m.LeagueName,
		LeagueCode:  m.LeagueCode,
		Position:    m.Position,
		DateOfBirth: m.DateOfBirth,
		Nationality: m.Nationality,
		ShirtNumber: m.ShirtNumber,
		Active:      m.Active,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}
