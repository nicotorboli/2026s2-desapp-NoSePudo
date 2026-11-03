package model

import "time"

// Límites del intervalo de sincronización. Viven acá y no en el service para
// que el DTO valide la entrada y el service la vuelva a validar sin duplicar
// la regla en dos paquetes.
const (
	MinSyncIntervalMinutes = 15
	MaxSyncIntervalMinutes = 30 * 24 * 60 // 30 días
)

// SyncType identifica qué sincronización ejecuta un run o un schedule.
type SyncType string

const (
	SyncTypePlayers    SyncType = "players"
	SyncTypeStatistics SyncType = "statistics"
)

// ParseSyncType traduce la forma textual que llega por la ruta. Devuelve false
// para lo que no reconoce; el valor cero nunca es válido.
func ParseSyncType(s string) (SyncType, bool) {
	switch SyncType(s) {
	case SyncTypePlayers, SyncTypeStatistics:
		return SyncType(s), true
	default:
		return "", false
	}
}

// Valid responde si el tipo es uno de los soportados.
func (t SyncType) Valid() bool {
	return t == SyncTypePlayers || t == SyncTypeStatistics
}

// SyncSchedule es la configuración de frecuencia de una sincronización.
type SyncSchedule struct {
	UpdatedAt       time.Time
	NextRunAt       time.Time
	UpdatedBy       *int64
	Type            SyncType
	IntervalMinutes int
	Enabled         bool
}
