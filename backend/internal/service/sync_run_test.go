package service_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type fakePlayerRunner struct {
	fn func(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)
}

func (f *fakePlayerRunner) SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
	if f.fn != nil {
		return f.fn(ctx, actor)
	}
	return dto.SyncPlayersResponse{}, nil
}

type fakeStatisticsRunner struct {
	fn func(ctx context.Context) (dto.SyncStatisticsResponse, error)
}

func (f *fakeStatisticsRunner) SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error) {
	if f.fn != nil {
		return f.fn(ctx)
	}
	return dto.SyncStatisticsResponse{}, nil
}

type syncServiceFixture struct {
	svc     *service.SyncRunService
	db      *sql.DB
	repos   *repository.Container
	players *fakePlayerRunner
	stats   *fakeStatisticsRunner
	actorID int64
}

func seedSyncServiceUser(t *testing.T, db *sql.DB, email string) int64 {
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

func newSyncServiceFixture(t *testing.T) *syncServiceFixture {
	t.Helper()

	db := startStatisticsPostgres(t)
	repos := repository.NewContainer(db, dao.NewContainer(db))
	actorID := seedSyncServiceUser(t, db, "sync-runs-service@nosepudo.ar")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	players := &fakePlayerRunner{}
	stats := &fakeStatisticsRunner{}

	svc := service.NewSyncRunService(
		t.Context(),
		30*time.Minute,
		players,
		stats,
		repos.SyncRun,
		repos.SyncSchedule,
		logger,
	)

	return &syncServiceFixture{
		svc:     svc,
		db:      db,
		repos:   repos,
		actorID: actorID,
		players: players,
		stats:   stats,
	}
}

func (f *syncServiceFixture) wait(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.svc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestSyncRunServiceTriggerManualPlayers(t *testing.T) {
	f := newSyncServiceFixture(t)

	var actor string
	f.players.fn = func(_ context.Context, gotActor string) (dto.SyncPlayersResponse, error) {
		actor = gotActor
		return dto.SyncPlayersResponse{
			Status:           "success",
			Message:          "ok",
			TotalProcessed:   10,
			TotalUpdated:     8,
			TotalDeactivated: 2,
		}, nil
	}

	run, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
	if err != nil {
		t.Fatalf("Trigger devolvió error: %v", err)
	}
	if run.Status != "running" || run.Trigger != "manual" {
		t.Fatalf("run inesperado: %+v", run)
	}

	f.wait(t)

	finished, err := f.svc.GetRun(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if finished.Status != "succeeded" || finished.TotalProcessed != 10 || finished.TotalUpdated != 8 {
		t.Errorf("run finalizado inesperado: %+v", finished)
	}
	if finished.Progress != nil {
		t.Error("el progreso debería haberse descartado al terminar")
	}
	if actor != "user/"+strconv.FormatInt(f.actorID, 10) {
		t.Errorf("actor = %q", actor)
	}
}

func TestSyncRunServiceTriggerCronUsesSystemActor(t *testing.T) {
	f := newSyncServiceFixture(t)

	var actor string
	f.players.fn = func(_ context.Context, gotActor string) (dto.SyncPlayersResponse, error) {
		actor = gotActor
		return dto.SyncPlayersResponse{Status: "success"}, nil
	}

	run, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, nil)
	if err != nil {
		t.Fatalf("Trigger devolvió error: %v", err)
	}
	if run.Trigger != "cron" {
		t.Errorf("trigger = %q, se esperaba cron", run.Trigger)
	}

	f.wait(t)

	if actor != "system/cron" {
		t.Errorf("actor = %q, se esperaba system/cron", actor)
	}
}

func TestSyncRunServiceSkipsWhenAnotherRunIsInProgress(t *testing.T) {
	f := newSyncServiceFixture(t)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f.players.fn = func(ctx context.Context, _ string) (dto.SyncPlayersResponse, error) {
		once.Do(func() { close(started) })
		select {
		case <-release:
			return dto.SyncPlayersResponse{Status: "success"}, nil
		case <-ctx.Done():
			return dto.SyncPlayersResponse{}, ctx.Err()
		}
	}

	first, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
	if err != nil {
		t.Fatalf("primer Trigger: %v", err)
	}
	<-started

	second, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
	if err != nil {
		t.Fatalf("segundo Trigger: %v", err)
	}
	if second.Status != "skipped" {
		t.Fatalf("el segundo run quedó %q, se esperaba skipped", second.Status)
	}
	if second.Message == "" {
		t.Error("el run omitido no explica el motivo")
	}

	close(release)
	f.wait(t)

	finished, err := f.svc.GetRun(t.Context(), first.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if finished.Status != "succeeded" {
		t.Errorf("el primer run quedó %q", finished.Status)
	}
}

func TestSyncRunServiceProgressIsEphemeral(t *testing.T) {
	f := newSyncServiceFixture(t)

	reported := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f.players.fn = func(ctx context.Context, _ string) (dto.SyncPlayersResponse, error) {
		if reporter, ok := service.FromContext(ctx); ok {
			reporter(service.Progress{Current: 2, Total: 5, Message: "Sincronizando BL1"})
		}
		once.Do(func() { close(reported) })
		select {
		case <-release:
			return dto.SyncPlayersResponse{Status: "success"}, nil
		case <-ctx.Done():
			return dto.SyncPlayersResponse{}, ctx.Err()
		}
	}

	run, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	<-reported

	running, err := f.svc.GetRun(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if running.Progress == nil {
		t.Fatal("el progreso no se expuso mientras el run estaba en curso")
	}
	if running.Progress.Current != 2 || running.Progress.Total != 5 || running.Progress.Message != "Sincronizando BL1" {
		t.Errorf("progreso inesperado: %+v", running.Progress)
	}

	close(release)
	f.wait(t)

	finished, err := f.svc.GetRun(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("GetRun final: %v", err)
	}
	if finished.Progress != nil {
		t.Error("el progreso debería haberse descartado")
	}
}

func TestSyncRunServiceMapsPartialAndFailure(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		f := newSyncServiceFixture(t)
		f.players.fn = func(context.Context, string) (dto.SyncPlayersResponse, error) {
			return dto.SyncPlayersResponse{Status: "partial_success", Message: "parcial"}, nil
		}

		run, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
		if err != nil {
			t.Fatalf("Trigger: %v", err)
		}
		f.wait(t)

		finished, err := f.svc.GetRun(t.Context(), run.ID)
		if err != nil {
			t.Fatalf("GetRun: %v", err)
		}
		if finished.Status != "partial" {
			t.Errorf("status = %q, se esperaba partial", finished.Status)
		}
	})

	t.Run("failure", func(t *testing.T) {
		f := newSyncServiceFixture(t)
		f.players.fn = func(context.Context, string) (dto.SyncPlayersResponse, error) {
			return dto.SyncPlayersResponse{}, service.ErrRateLimitExceeded
		}

		run, err := f.svc.Trigger(t.Context(), model.SyncTypePlayers, &f.actorID)
		if err != nil {
			t.Fatalf("Trigger: %v", err)
		}
		f.wait(t)

		finished, err := f.svc.GetRun(t.Context(), run.ID)
		if err != nil {
			t.Fatalf("GetRun: %v", err)
		}
		if finished.Status != "failed" {
			t.Errorf("status = %q, se esperaba failed", finished.Status)
		}
		if finished.Message == "" || finished.Error == "" {
			t.Errorf("el run fallido no dejó diagnóstico: %+v", finished)
		}
	})
}

func TestSyncRunServiceUpdateSchedule(t *testing.T) {
	f := newSyncServiceFixture(t)

	if err := f.repos.SyncSchedule.Ensure(t.Context(), model.SyncSchedule{
		Type:            model.SyncTypePlayers,
		Enabled:         false,
		IntervalMinutes: 1440,
		NextRunAt:       time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	updated, err := f.svc.UpdateSchedule(t.Context(), model.SyncTypePlayers, true, 60, f.actorID)
	if err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}
	if !updated.Enabled || updated.IntervalMinutes != 60 {
		t.Errorf("schedule inesperado: %+v", updated)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != f.actorID {
		t.Errorf("updatedBy = %v", updated.UpdatedBy)
	}

	if _, err := f.svc.UpdateSchedule(t.Context(), model.SyncTypePlayers, true, 1, f.actorID); err == nil {
		t.Error("UpdateSchedule aceptó un intervalo menor al mínimo")
	}
	if _, err := f.svc.UpdateSchedule(t.Context(), model.SyncTypeStatistics, true, 60, f.actorID); !errors.Is(err, service.ErrSyncScheduleNotFound) {
		t.Errorf("esperaba ErrSyncScheduleNotFound, obtuve %v", err)
	}
	if _, err := f.svc.UpdateSchedule(t.Context(), model.SyncType("otro"), true, 60, f.actorID); !errors.Is(err, service.ErrInvalidSyncType) {
		t.Errorf("esperaba ErrInvalidSyncType, obtuve %v", err)
	}
}

func TestSyncRunServiceListSchedulesAndRuns(t *testing.T) {
	f := newSyncServiceFixture(t)

	for _, typ := range []model.SyncType{model.SyncTypePlayers, model.SyncTypeStatistics} {
		if err := f.repos.SyncSchedule.Ensure(t.Context(), model.SyncSchedule{
			Type:            typ,
			Enabled:         true,
			IntervalMinutes: 1440,
			NextRunAt:       time.Now().Add(24 * time.Hour),
		}); err != nil {
			t.Fatalf("Ensure(%s): %v", typ, err)
		}
	}

	schedules, err := f.svc.ListSchedules(t.Context())
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(schedules) != 2 {
		t.Errorf("ListSchedules devolvió %d, se esperaban 2", len(schedules))
	}

	for i := 0; i < 2; i++ {
		if finishErr := f.repos.SyncRun.Finish(t.Context(), mustStartRun(t, f, model.SyncTypePlayers), model.SyncRunResult{
			FinishedAt: time.Now(),
			Status:     model.SyncStatusSucceeded,
		}); finishErr != nil {
			t.Fatalf("Finish: %v", finishErr)
		}
	}

	playersType := model.SyncTypePlayers
	page, err := f.svc.ListRuns(t.Context(), &playersType, 0, 1)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 {
		t.Errorf("ListRuns devolvió total=%d, len=%d", page.Total, len(page.Items))
	}
}

func mustStartRun(t *testing.T, f *syncServiceFixture, typ model.SyncType) int64 {
	t.Helper()

	run, err := f.repos.SyncRun.Start(t.Context(), typ, model.SyncTriggerCron, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if run.Status != model.SyncStatusRunning {
		t.Fatalf("el run sembrado quedó %q", run.Status)
	}
	return run.ID
}

func TestSyncRunServiceRecoverAndNotFound(t *testing.T) {
	f := newSyncServiceFixture(t)

	run, err := f.repos.SyncRun.Start(t.Context(), model.SyncTypeStatistics, model.SyncTriggerCron, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	affected, err := f.svc.Recover(t.Context())
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if affected != 1 {
		t.Errorf("Recover afectó %d filas", affected)
	}

	recovered, err := f.svc.GetRun(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if recovered.Status != "failed" {
		t.Errorf("el run recuperado quedó %q", recovered.Status)
	}

	if _, err := f.svc.GetRun(t.Context(), 999999); !errors.Is(err, service.ErrSyncRunNotFound) {
		t.Errorf("esperaba ErrSyncRunNotFound, obtuve %v", err)
	}
}
