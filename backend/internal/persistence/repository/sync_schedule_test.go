package repository_test

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func TestSyncScheduleRepositoryEnsureAndSet(t *testing.T) {
	db := startPostgres(t)
	repo := repository.NewSyncScheduleRepository(dao.NewSyncScheduleDao(db))
	actorID := seedSyncUser(t, db, "sync-schedule@nosepudo.ar")
	now := time.Now().UTC().Truncate(time.Second)

	for _, schedule := range []model.SyncSchedule{
		{
			Type:            model.SyncTypePlayers,
			Enabled:         true,
			IntervalMinutes: 1440,
			NextRunAt:       now.Add(24 * time.Hour),
		},
		{
			Type:            model.SyncTypeStatistics,
			Enabled:         true,
			IntervalMinutes: 1440,
			NextRunAt:       now.Add(24 * time.Hour),
		},
	} {
		if err := repo.Ensure(t.Context(), schedule); err != nil {
			t.Fatalf("Ensure(%s): %v", schedule.Type, err)
		}
	}

	schedules, err := repo.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(schedules) != 2 {
		t.Fatalf("List devolvió %d schedules, se esperaban 2", len(schedules))
	}

	updated, found, err := repo.Set(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         false,
		IntervalMinutes: 60,
		NextRunAt:       now.Add(-time.Minute),
		UpdatedBy:       &actorID,
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !found || updated.Enabled || updated.IntervalMinutes != 60 {
		t.Fatalf("Set no aplicó los cambios: %+v", updated)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != actorID {
		t.Errorf("Set no guardó el actor: %+v", updated.UpdatedBy)
	}
}

func TestSyncScheduleRepositoryReserveDue(t *testing.T) {
	db := startPostgres(t)
	repo := repository.NewSyncScheduleRepository(dao.NewSyncScheduleDao(db))
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         false,
		IntervalMinutes: 15,
		NextRunAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Ensure players: %v", err)
	}
	if err := repo.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypeStatistics,
		Enabled:         true,
		IntervalMinutes: 15,
		NextRunAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Ensure statistics: %v", err)
	}

	due, err := repo.ReserveDue(t.Context(), now)
	if err != nil {
		t.Fatalf("ReserveDue: %v", err)
	}
	if len(due) != 1 || due[0].Type != model.SyncTypeStatistics {
		t.Errorf("ReserveDue devolvió %+v, se esperaba solo statistics", due)
	}

	again, err := repo.ReserveDue(t.Context(), now)
	if err != nil {
		t.Fatalf("el segundo ReserveDue: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("el segundo ReserveDue volvió a reservar: %+v", again)
	}
}
