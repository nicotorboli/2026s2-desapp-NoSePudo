package dto

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type Player struct {
	name     string
	position int8
}

func FromModel(p model.Player) Player {
	return Player{
		name:     p.Name,
		position: p.Position,
	}
}
