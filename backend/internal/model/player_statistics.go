package model

import (
	"time"
)

// PlayerStatistics es la fotografía de las estadísticas de un jugador tomada
// en una pasada del scraper de WhoScored. Es una tabla de solo anexado: cada
// sincronización inserta filas nuevas y nunca se actualiza una existente, de
// modo que el historial queda trazado por created_at.
type PlayerStatistics struct {
	CreatedAt time.Time
	Rating    float64
	PlayerID  int64
	Goals     int
	Assists   int
	Shots     int
	KeyPasses int
	Dribbles  int
	Tackles   int
}

// PlayerStatisticsRow es la fila cruda que devuelve el adaptador de WhoScored
// antes del matcheo contra los jugadores del catálogo.
type PlayerStatisticsRow struct {
	Name      string
	ClubName  string
	Rating    float64
	Goals     int
	Assists   int
	Shots     int
	KeyPasses int
	Dribbles  int
	Tackles   int
}
