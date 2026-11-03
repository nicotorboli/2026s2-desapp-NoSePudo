package dao_test

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestSyncScheduleSqlEnsureIsIdempotent(t *testing.T) {
	db := startPostgres(t)
	schedules := dao.NewSyncScheduleDao(db)
	now := time.Now().UTC().Truncate(time.Second)

	original := model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         true,
		IntervalMinutes: 1440,
		NextRunAt:       now.Add(24 * time.Hour),
	}
	if err := schedules.Ensure(t.Context(), original); err != nil {
		t.Fatalf("Ensure devolvió error: %v", err)
	}

	replacement := original
	replacement.IntervalMinutes = 15
	replacement.Enabled = false
	if err := schedules.Ensure(t.Context(), replacement); err != nil {
		t.Fatalf("el segundo Ensure devolvió error: %v", err)
	}

	got, found, err := schedules.Get(t.Context(), model.SyncTypePlayers)
	if err != nil {
		t.Fatalf("Get devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Get no encontró el schedule sembrado")
	}
	if got.IntervalMinutes != 1440 || !got.Enabled {
		t.Errorf("Ensure pisó la fila existente: %+v", got)
	}

	list, err := schedules.List(t.Context())
	if err != nil {
		t.Fatalf("List devolvió error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List devolvió %d schedules, se esperaba 1", len(list))
	}
}

func TestSyncScheduleSqlSetUpdatesAndReportsMissing(t *testing.T) {
	db := startPostgres(t)
	schedules := dao.NewSyncScheduleDao(db)
	actorID := seedUser(t, db, "schedules@nosepudo.ar")
	now := time.Now().UTC().Truncate(time.Second)

	if err := schedules.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         true,
		IntervalMinutes: 1440,
		NextRunAt:       now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("Ensure devolvió error: %v", err)
	}

	updated, found, err := schedules.Set(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         false,
		IntervalMinutes: 60,
		NextRunAt:       now.Add(time.Hour),
		UpdatedBy:       &actorID,
	})
	if err != nil {
		t.Fatalf("Set devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Set no encontró el schedule existente")
	}
	if updated.Enabled || updated.IntervalMinutes != 60 {
		t.Errorf("Set no aplicó los cambios: %+v", updated)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != actorID {
		t.Errorf("Set no guardó el actor: %+v", updated.UpdatedBy)
	}

	if _, found, err := schedules.Get(t.Context(), model.SyncTypeStatistics); err != nil {
		t.Fatalf("Get de un tipo inexistente devolvió error: %v", err)
	} else if found {
		t.Error("Get encontró un schedule que no existe")
	}

	if _, found, err := schedules.Set(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypeStatistics,
		Enabled:         true,
		IntervalMinutes: 60,
		NextRunAt:       now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Set de un tipo inexistente devolvió error: %v", err)
	} else if found {
		t.Error("Set actualizó un schedule que no existe")
	}
}

func TestSyncScheduleSqlReserveDueClaimsOnlyDueEnabled(t *testing.T) {
	db := startPostgres(t)
	schedules := dao.NewSyncScheduleDao(db)
	now := time.Now().UTC().Truncate(time.Second)

	if err := schedules.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         true,
		IntervalMinutes: 15,
		NextRunAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Ensure players: %v", err)
	}
	if err := schedules.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypeStatistics,
		Enabled:         false,
		IntervalMinutes: 15,
		NextRunAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Ensure statistics: %v", err)
	}

	due, err := schedules.ReserveDue(t.Context(), now)
	if err != nil {
		t.Fatalf("ReserveDue devolvió error: %v", err)
	}
	if len(due) != 1 || due[0].Type != model.SyncTypePlayers {
		t.Fatalf("ReserveDue devolvió %+v, se esperaba solo players", due)
	}

	advanced, _, err := schedules.Get(t.Context(), model.SyncTypePlayers)
	if err != nil {
		t.Fatalf("Get players: %v", err)
	}
	if !advanced.NextRunAt.After(now) {
		t.Errorf("next_run_at no avanzó: %v", advanced.NextRunAt)
	}

	again, err := schedules.ReserveDue(t.Context(), now)
	if err != nil {
		t.Fatalf("el segundo ReserveDue devolvió error: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("el segundo ReserveDue volvió a reservar: %+v", again)
	}
}

func TestSyncScheduleSqlEnableLegacyDefaults(t *testing.T) {
	db := startPostgres(t)
	schedules := dao.NewSyncScheduleDao(db)
	actorID := seedUser(t, db, "legacy@nosepudo.ar")
	now := time.Now().UTC().Truncate(time.Second)

	for _, schedule := range []model.SyncSchedule{
		{
			Type:            model.SyncTypePlayers,
			Enabled:         false,
			IntervalMinutes: 1440,
			NextRunAt:       now.Add(24 * time.Hour),
		},
		{
			Type:            model.SyncTypeStatistics,
			Enabled:         false,
			IntervalMinutes: 60,
			NextRunAt:       now.Add(time.Hour),
		},
	} {
		if err := schedules.Ensure(t.Context(), schedule); err != nil {
			t.Fatalf("Ensure(%s): %v", schedule.Type, err)
		}
	}

	if err := schedules.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncType("custom"),
		Enabled:         false,
		IntervalMinutes: 1440,
		NextRunAt:       now.Add(24 * time.Hour),
		UpdatedBy:       &actorID,
	}); err != nil {
		t.Fatalf("Ensure(custom): %v", err)
	}

	count, err := schedules.EnableLegacyDefaults(t.Context(), now)
	if err != nil {
		t.Fatalf("EnableLegacyDefaults: %v", err)
	}
	if count != 1 {
		t.Errorf("EnableLegacyDefaults afectó %d filas, se esperaba 1", count)
	}

	players, _, err := schedules.Get(t.Context(), model.SyncTypePlayers)
	if err != nil {
		t.Fatalf("Get players: %v", err)
	}
	if !players.Enabled {
		t.Error("el schedule legacy no quedó habilitado")
	}

	statistics, _, err := schedules.Get(t.Context(), model.SyncTypeStatistics)
	if err != nil {
		t.Fatalf("Get statistics: %v", err)
	}
	if statistics.Enabled {
		t.Error("EnableLegacyDefaults pisó un intervalo configurado")
	}

	custom, _, err := schedules.Get(t.Context(), model.SyncType("custom"))
	if err != nil {
		t.Fatalf("Get custom: %v", err)
	}
	if custom.Enabled {
		t.Error("EnableLegacyDefaults pisó una configuración del operador")
	}
}
