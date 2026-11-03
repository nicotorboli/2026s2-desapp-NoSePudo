package dao_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func insertRunningRun(t *testing.T, db *sql.DB, runs *dao.SyncRunSql, run model.SyncRun) int64 {
	t.Helper()

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	id, started, err := runs.InsertRunning(t.Context(), tx, run)
	if err != nil {
		t.Fatalf("InsertRunning: %v", err)
	}
	if !started {
		t.Fatal("InsertRunning no arrancó el run esperado")
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	return id
}

func insertSkippedRun(t *testing.T, db *sql.DB, runs *dao.SyncRunSql, run model.SyncRun) int64 {
	t.Helper()

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	id, err := runs.InsertSkipped(t.Context(), tx, run)
	if err != nil {
		t.Fatalf("InsertSkipped: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	return id
}

func TestSyncRunSqlInsertRunningRejectsAConcurrentRun(t *testing.T) {
	db := startPostgres(t)
	runs := dao.NewSyncRunDao(db)
	actorID := seedUser(t, db)
	now := time.Now().UTC()

	firstID := insertRunningRun(t, db, runs, model.SyncRun{
		Type:        model.SyncTypePlayers,
		Trigger:     model.SyncTriggerManual,
		Status:      model.SyncStatusRunning,
		StartedAt:   now,
		TriggeredBy: &actorID,
	})

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, started, runErr := runs.InsertRunning(t.Context(), tx, model.SyncRun{
		Type:      model.SyncTypePlayers,
		Trigger:   model.SyncTriggerCron,
		Status:    model.SyncStatusRunning,
		StartedAt: now.Add(time.Second),
	}); runErr != nil {
		t.Fatalf("el segundo InsertRunning devolvió error: %v", runErr)
	} else if started {
		t.Fatal("InsertRunning permitió un segundo run concurrente del mismo tipo")
	}

	run, found, err := runs.Get(t.Context(), firstID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get no encontró el run")
	}
	if run.Status != model.SyncStatusRunning || run.Trigger != model.SyncTriggerManual {
		t.Errorf("run inesperado: %+v", run)
	}

	affected, err := runs.MarkInterrupted(t.Context())
	if err != nil {
		t.Fatalf("MarkInterrupted: %v", err)
	}
	if affected != 1 {
		t.Errorf("MarkInterrupted afectó %d filas, se esperaba 1", affected)
	}

	run, _, err = runs.Get(t.Context(), firstID)
	if err != nil {
		t.Fatalf("Get después de MarkInterrupted: %v", err)
	}
	if run.Status != model.SyncStatusFailed {
		t.Errorf("el run interrumpido quedó %q, se esperaba failed", run.Status)
	}
}

func TestSyncRunSqlFinishPersistsCounters(t *testing.T) {
	db := startPostgres(t)
	runs := dao.NewSyncRunDao(db)
	now := time.Now().UTC()

	id := insertRunningRun(t, db, runs, model.SyncRun{
		Type:      model.SyncTypeStatistics,
		Trigger:   model.SyncTriggerCron,
		Status:    model.SyncStatusRunning,
		StartedAt: now,
	})

	finishedAt := now.Add(2 * time.Minute).Truncate(time.Microsecond)
	if err := runs.Finish(t.Context(), id, model.SyncRunResult{
		FinishedAt:       finishedAt,
		Status:           model.SyncStatusPartial,
		Message:          "parcial",
		Error:            "rate limit",
		TotalProcessed:   10,
		TotalMatched:     8,
		TotalUnmatched:   2,
		TotalInserted:    8,
		TotalUpdated:     0,
		TotalDeactivated: 0,
	}); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	run, found, err := runs.Get(t.Context(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get no encontró el run")
	}
	if run.Status != model.SyncStatusPartial || run.Error != "rate limit" {
		t.Errorf("run inesperado: %+v", run)
	}
	if run.TotalProcessed != 10 || run.TotalMatched != 8 || run.TotalUnmatched != 2 || run.TotalInserted != 8 {
		t.Errorf("contadores inesperados: %+v", run)
	}
	if run.FinishedAt == nil || !run.FinishedAt.Equal(finishedAt) {
		t.Errorf("finishedAt = %v, se esperaba %v", run.FinishedAt, finishedAt)
	}
}

func TestSyncRunSqlInsertSkippedStoresTheReason(t *testing.T) {
	db := startPostgres(t)
	runs := dao.NewSyncRunDao(db)
	now := time.Now().UTC()

	id := insertSkippedRun(t, db, runs, model.SyncRun{
		Type:       model.SyncTypePlayers,
		Trigger:    model.SyncTriggerCron,
		Status:     model.SyncStatusSkipped,
		StartedAt:  now,
		FinishedAt: &now,
		Message:    "ya hay una sincronización en curso",
	})

	run, found, err := runs.Get(t.Context(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get no encontró el run omitido")
	}
	if run.Status != model.SyncStatusSkipped || run.Message != "ya hay una sincronización en curso" {
		t.Errorf("run omitido inesperado: %+v", run)
	}
	if run.FinishedAt == nil {
		t.Error("el run omitido no tiene finished_at")
	}
}

func TestSyncRunSqlListFiltersAndCounts(t *testing.T) {
	db := startPostgres(t)
	runs := dao.NewSyncRunDao(db)
	base := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 2; i++ {
		id := insertRunningRun(t, db, runs, model.SyncRun{
			Type:      model.SyncTypePlayers,
			Trigger:   model.SyncTriggerManual,
			Status:    model.SyncStatusRunning,
			StartedAt: base.Add(time.Duration(i) * time.Minute),
		})
		if err := runs.Finish(t.Context(), id, model.SyncRunResult{
			FinishedAt: base.Add(time.Duration(i)*time.Minute + 30*time.Second),
			Status:     model.SyncStatusSucceeded,
		}); err != nil {
			t.Fatalf("Finish: %v", err)
		}
	}

	statsID := insertRunningRun(t, db, runs, model.SyncRun{
		Type:      model.SyncTypeStatistics,
		Trigger:   model.SyncTriggerCron,
		Status:    model.SyncStatusRunning,
		StartedAt: base.Add(3 * time.Minute),
	})
	if err := runs.Finish(t.Context(), statsID, model.SyncRunResult{
		FinishedAt: base.Add(4 * time.Minute),
		Status:     model.SyncStatusSucceeded,
	}); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	all, total, err := runs.List(t.Context(), nil, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("List devolvió total=%d, len=%d; se esperaban 3", total, len(all))
	}
	if all[0].Type != model.SyncTypeStatistics {
		t.Errorf("List no ordenó por started_at DESC: %+v", all)
	}

	playersType := model.SyncTypePlayers
	players, total, err := runs.List(t.Context(), &playersType, 0, 10)
	if err != nil {
		t.Fatalf("List players: %v", err)
	}
	if total != 2 || len(players) != 2 {
		t.Fatalf("List players devolvió total=%d, len=%d; se esperaban 2", total, len(players))
	}

	page, _, err := runs.List(t.Context(), nil, 1, 1)
	if err != nil {
		t.Fatalf("List paginado: %v", err)
	}
	if len(page) != 1 || page[0].Type != model.SyncTypePlayers {
		t.Errorf("la segunda página no trajo el run esperado: %+v", page)
	}
}

func TestSyncRunSqlGetMissing(t *testing.T) {
	db := startPostgres(t)
	runs := dao.NewSyncRunDao(db)

	if _, found, err := runs.Get(t.Context(), 999999); err != nil {
		t.Fatalf("Get de un id inexistente devolvió error: %v", err)
	} else if found {
		t.Error("Get encontró un run que no existe")
	}
}
