package dto

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type Player struct {
	Name string `json:"name"`
	Position int8 `json:"position"`
}

func DesdeModelo(p model.Player) Player {
	return Player{
		Name: p.Name,
		Position: p.Position,
	}
}

