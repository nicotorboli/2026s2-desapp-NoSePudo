package service_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func TestSyncSchedulerStartEnsuresDefaultSchedules(t *testing.T) {
	f := newSyncServiceFixture(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Fila con el default viejo: deshabilitada, diaria y sin actor. El scheduler
	// debe encenderla al arrancar sin tocar configuraciones explícitas.
	if err := f.repos.SyncSchedule.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         false,
		IntervalMinutes: 1440,
		NextRunAt:       time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("Ensure legacy: %v", err)
	}

	scheduler := service.NewSyncScheduler(f.svc, f.repos.SyncSchedule, service.DefaultSyncSettings(), logger)
	scheduler.Start(t.Context())
	scheduler.Stop()
	f.wait(t)

	schedules, err := f.svc.ListSchedules(t.Context())
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(schedules) != 2 {
		t.Fatalf("ListSchedules devolvió %d, se esperaban 2", len(schedules))
	}

	for _, schedule := range schedules {
		if !schedule.Enabled {
			t.Errorf("el schedule %s no quedó habilitado por defecto", schedule.Type)
		}
		if schedule.IntervalMinutes != 1440 {
			t.Errorf("el schedule %s quedó con intervalo %d", schedule.Type, schedule.IntervalMinutes)
		}
	}
}

func TestSyncSchedulerStartTriggersCatchUp(t *testing.T) {
	f := newSyncServiceFixture(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := f.repos.SyncSchedule.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         true,
		IntervalMinutes: 15,
		NextRunAt:       time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Ensure players: %v", err)
	}

	called := make(chan struct{})
	var once sync.Once
	f.players.fn = func(context.Context, string) (dto.SyncPlayersResponse, error) {
		once.Do(func() { close(called) })
		return dto.SyncPlayersResponse{Status: "success", TotalProcessed: 3}, nil
	}

	scheduler := service.NewSyncScheduler(f.svc, f.repos.SyncSchedule, service.DefaultSyncSettings(), logger)
	scheduler.Start(t.Context())

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("el scheduler no disparó la sincronización vencida")
	}

	scheduler.Stop()
	f.wait(t)

	schedules, err := f.svc.ListSchedules(t.Context())
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	for _, schedule := range schedules {
		if schedule.Type == "players" && schedule.IntervalMinutes != 15 {
			t.Errorf("el schedule existente fue pisado: intervalo %d", schedule.IntervalMinutes)
		}
	}

	runs, err := f.svc.ListRuns(t.Context(), nil, 0, 10)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if runs.Total != 1 {
		t.Fatalf("ListRuns devolvió %d runs, se esperaba 1", runs.Total)
	}
	if runs.Items[0].Trigger != "cron" || runs.Items[0].Status != "succeeded" {
		t.Errorf("run del scheduler inesperado: %+v", runs.Items[0])
	}
	if runs.Items[0].Type != "players" || runs.Items[0].TotalProcessed != 3 {
		t.Errorf("run del scheduler inesperado: %+v", runs.Items[0])
	}
}
