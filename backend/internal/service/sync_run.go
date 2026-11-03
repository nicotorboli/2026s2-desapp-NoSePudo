package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var (
	ErrSyncRunNotFound      = errors.New("sync run not found")
	ErrSyncScheduleNotFound = errors.New("sync schedule not found")
	ErrInvalidSyncType      = errors.New("invalid sync type")
)

// finishTimeout es cuánto espera un run para persistir su resultado final. Es
// independiente del timeout de ejecución: si un run se cortó por timeout, su
// fila igual tiene que quedar cerrada.
const finishTimeout = 5 * time.Second

type PlayerSyncRunner interface {
	SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)
}

type StatisticsSyncRunner interface {
	SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error)
}

type SyncRunRepository interface {
	Start(ctx context.Context, typ model.SyncType, trigger model.SyncTrigger, actorID *int64) (model.SyncRun, error)
	Finish(ctx context.Context, id int64, result model.SyncRunResult) error
	Get(ctx context.Context, id int64) (model.SyncRun, bool, error)
	List(ctx context.Context, typ *model.SyncType, offset, limit int) ([]model.SyncRun, int, error)
	MarkInterrupted(ctx context.Context) (int64, error)
}

type SyncScheduleRepository interface {
	Ensure(ctx context.Context, schedule model.SyncSchedule) error
	List(ctx context.Context) ([]model.SyncSchedule, error)
	Get(ctx context.Context, typ model.SyncType) (model.SyncSchedule, bool, error)
	Set(ctx context.Context, schedule model.SyncSchedule) (model.SyncSchedule, bool, error)
	ReserveDue(ctx context.Context, now time.Time) ([]model.SyncSchedule, error)
}

// SyncRunService coordina las ejecuciones: crea el run, lo corre en segundo
// plano y expone su estado y su progreso efímero.
type SyncRunService struct {
	runCtx     context.Context
	players    PlayerSyncRunner
	statistics StatisticsSyncRunner
	runs       SyncRunRepository
	schedules  SyncScheduleRepository
	logger     *slog.Logger
	progress   map[int64]dto.SyncRunProgress
	wg         sync.WaitGroup
	runTimeout time.Duration
	mu         sync.Mutex
}

func NewSyncRunService(
	runCtx context.Context,
	runTimeout time.Duration,
	players PlayerSyncRunner,
	statistics StatisticsSyncRunner,
	runs SyncRunRepository,
	schedules SyncScheduleRepository,
	logger *slog.Logger,
) *SyncRunService {
	if runTimeout <= 0 {
		runTimeout = 30 * time.Minute
	}

	return &SyncRunService{
		runCtx:     runCtx,
		runTimeout: runTimeout,
		players:    players,
		statistics: statistics,
		runs:       runs,
		schedules:  schedules,
		logger:     logger,
		progress:   make(map[int64]dto.SyncRunProgress),
	}
}

// Trigger crea el run y, si arranca, lo lanza en segundo plano. Un actorID nil
// significa cron; un actorID presente significa disparo manual.
func (s *SyncRunService) Trigger(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error) {
	if !typ.Valid() {
		return dto.SyncRunResponse{}, ErrInvalidSyncType
	}

	trigger := model.SyncTriggerManual
	if actorID == nil {
		trigger = model.SyncTriggerCron
	}

	run, err := s.runs.Start(ctx, typ, trigger, actorID)
	if err != nil {
		return dto.SyncRunResponse{}, fmt.Errorf("start sync run: %w", err)
	}

	if run.Status == model.SyncStatusRunning {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.execute(run)
		}()
	}

	return dto.SyncRunDesdeModelo(run, s.progressSnapshot(run.ID)), nil
}

// Recover marca como failed los runs que quedaron en curso tras un reinicio.
func (s *SyncRunService) Recover(ctx context.Context) (int64, error) {
	count, err := s.runs.MarkInterrupted(ctx)
	if err != nil {
		return 0, fmt.Errorf("recover interrupted sync runs: %w", err)
	}
	if count > 0 {
		s.logger.InfoContext(ctx, "marked interrupted sync runs as failed", "count", count)
	}
	return count, nil
}

func (s *SyncRunService) GetRun(ctx context.Context, id int64) (dto.SyncRunResponse, error) {
	run, found, err := s.runs.Get(ctx, id)
	if err != nil {
		return dto.SyncRunResponse{}, fmt.Errorf("get sync run: %w", err)
	}
	if !found {
		return dto.SyncRunResponse{}, ErrSyncRunNotFound
	}

	var progress *dto.SyncRunProgress
	if run.Status == model.SyncStatusRunning {
		progress = s.progressSnapshot(id)
	}

	return dto.SyncRunDesdeModelo(run, progress), nil
}

func (s *SyncRunService) ListRuns(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error) {
	runs, total, err := s.runs.List(ctx, typ, offset, limit)
	if err != nil {
		return dto.SyncRunPageResponse{}, fmt.Errorf("list sync runs: %w", err)
	}

	items := make([]dto.SyncRunResponse, 0, len(runs))
	for _, run := range runs {
		var progress *dto.SyncRunProgress
		if run.Status == model.SyncStatusRunning {
			progress = s.progressSnapshot(run.ID)
		}
		items = append(items, dto.SyncRunDesdeModelo(run, progress))
	}

	return dto.SyncRunPageResponse{Items: items, Offset: offset, Limit: limit, Total: total}, nil
}

func (s *SyncRunService) ListSchedules(ctx context.Context) ([]dto.SyncScheduleResponse, error) {
	schedules, err := s.schedules.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sync schedules: %w", err)
	}

	items := make([]dto.SyncScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		items = append(items, dto.SyncScheduleDesdeModelo(schedule))
	}

	return items, nil
}

func (s *SyncRunService) UpdateSchedule(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error) {
	if !typ.Valid() {
		return dto.SyncScheduleResponse{}, ErrInvalidSyncType
	}
	if intervalMinutes < model.MinSyncIntervalMinutes || intervalMinutes > model.MaxSyncIntervalMinutes {
		return dto.SyncScheduleResponse{}, fmt.Errorf("interval minutes out of range")
	}

	schedule := model.SyncSchedule{
		Type:            typ,
		Enabled:         enabled,
		IntervalMinutes: intervalMinutes,
		NextRunAt:       time.Now().Add(time.Duration(intervalMinutes) * time.Minute),
		UpdatedBy:       &actorID,
	}

	updated, found, err := s.schedules.Set(ctx, schedule)
	if err != nil {
		return dto.SyncScheduleResponse{}, fmt.Errorf("update sync schedule: %w", err)
	}
	if !found {
		return dto.SyncScheduleResponse{}, ErrSyncScheduleNotFound
	}

	return dto.SyncScheduleDesdeModelo(updated), nil
}

// Shutdown espera a que terminen los runs en curso, o al contexto de apagado.
func (s *SyncRunService) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *SyncRunService) execute(run model.SyncRun) {
	ctx, cancel := context.WithTimeout(s.runCtx, s.runTimeout)
	defer cancel()
	s.run(ctx, run)
}

func (s *SyncRunService) run(ctx context.Context, run model.SyncRun) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.ErrorContext(ctx, "panic during sync run", "run_id", run.ID, "panic", recovered)
			s.finish(run.ID, model.SyncRunResult{
				Status:     model.SyncStatusFailed,
				FinishedAt: time.Now(),
				Message:    "synchronization panicked",
				Error:      fmt.Sprintf("%v", recovered),
			})
		}
	}()

	ctx = WithProgress(ctx, func(p Progress) {
		s.setProgress(run.ID, p)
	})

	result, err := s.dispatch(ctx, run)
	if err != nil {
		result = failureResult(err)
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now()
	}

	s.finish(run.ID, result)
}

func (s *SyncRunService) dispatch(ctx context.Context, run model.SyncRun) (model.SyncRunResult, error) {
	switch run.Type {
	case model.SyncTypePlayers:
		return s.runPlayers(ctx, run)
	case model.SyncTypeStatistics:
		return s.runStatistics(ctx, run)
	default:
		return model.SyncRunResult{}, ErrInvalidSyncType
	}
}

func (s *SyncRunService) runPlayers(ctx context.Context, run model.SyncRun) (model.SyncRunResult, error) {
	actor := "system/cron"
	if run.TriggeredBy != nil {
		actor = fmt.Sprintf("user/%d", *run.TriggeredBy)
	}

	resp, err := s.players.SyncPlayers(ctx, actor)
	if err != nil {
		return model.SyncRunResult{}, err
	}

	status := model.SyncStatusSucceeded
	if resp.Status == "partial_success" {
		status = model.SyncStatusPartial
	}

	return model.SyncRunResult{
		Status:           status,
		Message:          resp.Message,
		TotalProcessed:   resp.TotalProcessed,
		TotalUpdated:     resp.TotalUpdated,
		TotalDeactivated: resp.TotalDeactivated,
	}, nil
}

func (s *SyncRunService) runStatistics(ctx context.Context, run model.SyncRun) (model.SyncRunResult, error) {
	resp, err := s.statistics.SyncStatistics(ctx)
	if err != nil {
		return model.SyncRunResult{}, err
	}

	status := model.SyncStatusSucceeded
	if resp.Status == "partial_success" {
		status = model.SyncStatusPartial
	}

	return model.SyncRunResult{
		Status:         status,
		Message:        resp.Message,
		TotalProcessed: resp.TotalProcessed,
		TotalMatched:   resp.TotalMatched,
		TotalUnmatched: resp.TotalUnmatched,
		TotalInserted:  resp.TotalInserted,
	}, nil
}

func (s *SyncRunService) finish(id int64, result model.SyncRunResult) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(s.runCtx), finishTimeout)
	defer cancel()

	if err := s.runs.Finish(ctx, id, result); err != nil {
		s.logger.ErrorContext(ctx, "failed to finish sync run", "run_id", id, "error", err.Error())
	}
	s.clearProgress(id)
}

func (s *SyncRunService) setProgress(id int64, p Progress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress[id] = dto.SyncRunProgress{Current: p.Current, Total: p.Total, Message: p.Message}
}

func (s *SyncRunService) progressSnapshot(id int64) *dto.SyncRunProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.progress[id]
	if !ok {
		return nil
	}
	return &p
}

func (s *SyncRunService) clearProgress(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.progress, id)
}

func failureResult(err error) model.SyncRunResult {
	message := "synchronization failed"
	switch {
	case errors.Is(err, ErrRateLimitExceeded):
		message = "external provider rate limit reached"
	case errors.Is(err, ErrStatisticsRateLimitExceeded):
		message = "statistics provider rate limit reached"
	case errors.Is(err, ErrStatisticsProviderBlocked):
		message = "statistics provider blocked the request"
	case errors.Is(err, context.DeadlineExceeded):
		message = "synchronization timed out"
	case errors.Is(err, context.Canceled):
		message = "synchronization was interrupted"
	}

	return model.SyncRunResult{
		Status:  model.SyncStatusFailed,
		Message: message,
		Error:   err.Error(),
	}
}
