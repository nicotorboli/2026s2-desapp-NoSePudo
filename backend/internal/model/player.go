package model

import (
	"time"
)

type Player struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DateOfBirth *string
	Nationality *string
	ShirtNumber *int
	Name        string
	ClubName    string
	LeagueName  string
	LeagueCode  string
	Position    string
	ID          int64
	ExternalID  int64
	Active      bool
}

type PlayerFilter struct {
	Search          string
	League          string
	Club            string
	Position        string
	Page            int
	Limit           int
	IncludeInactive bool
}

type PageResult[T any] struct {
	Items      []T
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}
