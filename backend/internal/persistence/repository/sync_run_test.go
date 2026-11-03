package repository_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func seedSyncUser(t *testing.T, db *sql.DB, email string) int64 {
	t.Helper()

	var id int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO users (email, password_hash, privilege) VALUES ($1, 'x', 2) RETURNING id`,
		email,
	).Scan(&id); err != nil {
		t.Fatalf("no se pudo sembrar el usuario: %v", err)
	}
	return id
}

func TestSyncRunRepositoryStartCreatesSkippedWhenOneIsRunning(t *testing.T) {
	db := startPostgres(t)
	repo := repository.NewSyncRunRepository(db, dao.NewSyncRunDao(db))
	actorID := seedSyncUser(t, db, "sync-run@nosepudo.ar")

	first, err := repo.Start(t.Context(), model.SyncTypePlayers, model.SyncTriggerManual, &actorID)
	if err != nil {
		t.Fatalf("el primer Start devolvió error: %v", err)
	}
	if first.Status != model.SyncStatusRunning {
		t.Fatalf("el primer run quedó %q, se esperaba running", first.Status)
	}
	if first.TriggeredBy == nil || *first.TriggeredBy != actorID {
		t.Errorf("el primer run no guardó el actor: %+v", first.TriggeredBy)
	}

	second, err := repo.Start(t.Context(), model.SyncTypePlayers, model.SyncTriggerCron, nil)
	if err != nil {
		t.Fatalf("el segundo Start devolvió error: %v", err)
	}
	if second.Status != model.SyncStatusSkipped {
		t.Fatalf("el segundo run quedó %q, se esperaba skipped", second.Status)
	}
	if second.Message == "" || second.FinishedAt == nil {
		t.Errorf("el run omitido no explica el motivo: %+v", second)
	}

	if finishErr := repo.Finish(t.Context(), first.ID, model.SyncRunResult{
		FinishedAt:     time.Now().UTC(),
		Status:         model.SyncStatusSucceeded,
		TotalProcessed: 5,
	}); finishErr != nil {
		t.Fatalf("Finish devolvió error: %v", finishErr)
	}

	third, err := repo.Start(t.Context(), model.SyncTypePlayers, model.SyncTriggerCron, nil)
	if err != nil {
		t.Fatalf("el tercer Start devolvió error: %v", err)
	}
	if third.Status != model.SyncStatusRunning {
		t.Errorf("después de terminar el primero, el tercero quedó %q", third.Status)
	}
}

func TestSyncRunRepositoryMarkInterrupted(t *testing.T) {
	db := startPostgres(t)
	repo := repository.NewSyncRunRepository(db, dao.NewSyncRunDao(db))

	run, err := repo.Start(t.Context(), model.SyncTypeStatistics, model.SyncTriggerCron, nil)
	if err != nil {
		t.Fatalf("Start devolvió error: %v", err)
	}

	affected, err := repo.MarkInterrupted(t.Context())
	if err != nil {
		t.Fatalf("MarkInterrupted devolvió error: %v", err)
	}
	if affected != 1 {
		t.Errorf("MarkInterrupted afectó %d filas, se esperaba 1", affected)
	}

	recovered, found, err := repo.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("Get devolvió error: %v", err)
	}
	if !found || recovered.Status != model.SyncStatusFailed {
		t.Errorf("el run no quedó failed: %+v", recovered)
	}
}
