package model

import "time"

// SyncTrigger distingue el origen de un run: lo disparó el scheduler o un
// operador a mano. Los dos terminan en el mismo SyncRun.
type SyncTrigger string

const (
	SyncTriggerCron   SyncTrigger = "cron"
	SyncTriggerManual SyncTrigger = "manual"
)

// SyncStatus es el ciclo de vida de un run.
type SyncStatus string

const (
	SyncStatusRunning   SyncStatus = "running"
	SyncStatusSucceeded SyncStatus = "succeeded"
	SyncStatusPartial   SyncStatus = "partial"
	SyncStatusFailed    SyncStatus = "failed"
	SyncStatusSkipped   SyncStatus = "skipped"
)

// SyncRun es la ejecución de una sincronización, disparada por cron o a mano.
// El progreso intermedio no vive acá: es efímero y lo maneja el service.
type SyncRun struct {
	StartedAt        time.Time
	TriggeredBy      *int64
	FinishedAt       *time.Time
	Status           SyncStatus
	Message          string
	Error            string
	Trigger          SyncTrigger
	Type             SyncType
	ID               int64
	TotalProcessed   int
	TotalUpdated     int
	TotalDeactivated int
	TotalMatched     int
	TotalUnmatched   int
	TotalInserted    int
}

// SyncRunResult es lo que el service calcula al terminar un run y entrega a la
// persistencia para cerrar la fila.
type SyncRunResult struct {
	FinishedAt       time.Time
	Status           SyncStatus
	Message          string
	Error            string
	TotalProcessed   int
	TotalUpdated     int
	TotalDeactivated int
	TotalMatched     int
	TotalUnmatched   int
	TotalInserted    int
}
