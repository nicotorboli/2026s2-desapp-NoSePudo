package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// schedulerTick es cada cuánto el scheduler mira si hay schedules vencidos.
// No es la frecuencia de ejecución: esa es interval_minutes. El tick solo fija
// el desfase máximo con que se detecta un schedule que venció.
const schedulerTick = time.Minute

// SyncSettings es la configuración inicial de las sincronizaciones. El service
// no importa configuration a propósito: main traduce Cfg a esto.
type SyncSettings struct {
	RunTimeout         time.Duration
	PlayersInterval    time.Duration
	StatisticsInterval time.Duration
	PlayersEnabled     bool
	StatisticsEnabled  bool
}

// DefaultSyncSettings deshabilita las sincronizaciones por defecto. La usan
// los tests y el camino que no recibe configuración explícita.
func DefaultSyncSettings() SyncSettings {
	return SyncSettings{
		RunTimeout:         30 * time.Minute,
		PlayersInterval:    24 * time.Hour,
		StatisticsInterval: 24 * time.Hour,
	}
}

// SyncScheduler despierta periódicamente, reserva los schedules vencidos y
// dispara el run correspondiente. Corre en una sola goroutine del proceso.
type SyncScheduler struct {
	schedules SyncScheduleRepository
	svc       *SyncRunService
	logger    *slog.Logger
	cancel    context.CancelFunc
	done      chan struct{}
	settings  SyncSettings
	tick      time.Duration
	mu        sync.Mutex
	started   bool
}

func NewSyncScheduler(svc *SyncRunService, schedules SyncScheduleRepository, settings SyncSettings, logger *slog.Logger) *SyncScheduler {
	return &SyncScheduler{
		svc:       svc,
		schedules: schedules,
		settings:  settings,
		logger:    logger,
		tick:      schedulerTick,
		done:      make(chan struct{}),
	}
}

func (s *SyncScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.cancel = cancel
	s.started = true
	s.mu.Unlock()

	if _, err := s.svc.Recover(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to recover interrupted sync runs", "error", err.Error())
	}
	if err := s.ensureSchedules(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to ensure sync schedules", "error", err.Error())
	}

	// Primer tick inmediato: si un schedule venció mientras el proceso estaba
	// apagado, se ejecuta acá (catch-up).
	s.tickOnce(ctx)

	go s.loop(ctx)
}

// Stop cancela el ticker y espera a que el loop termine, para que ningún tick
// en curso dispare un run después de que empiece el apagado.
func (s *SyncScheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	started := s.started
	s.mu.Unlock()

	if !started {
		return
	}
	if cancel != nil {
		cancel()
	}

	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

func (s *SyncScheduler) loop(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickOnce(ctx)
		}
	}
}

func (s *SyncScheduler) tickOnce(ctx context.Context) {
	due, err := s.schedules.ReserveDue(ctx, time.Now())
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to reserve due sync schedules", "error", err.Error())
		return
	}

	for _, schedule := range due {
		s.logger.InfoContext(ctx, "scheduled sync due", "type", schedule.Type, "next_run_at", schedule.NextRunAt)
		if _, err := s.svc.Trigger(ctx, schedule.Type, nil); err != nil {
			s.logger.ErrorContext(ctx, "failed to trigger scheduled sync", "type", schedule.Type, "error", err.Error())
		}
	}
}

func (s *SyncScheduler) ensureSchedules(ctx context.Context) error {
	now := time.Now()
	for _, schedule := range defaultSchedules(s.settings, now) {
		if err := s.schedules.Ensure(ctx, schedule); err != nil {
			return err
		}
	}
	return nil
}

func defaultSchedules(settings SyncSettings, now time.Time) []model.SyncSchedule {
	playersInterval := intervalMinutes(settings.PlayersInterval)
	statisticsInterval := intervalMinutes(settings.StatisticsInterval)

	return []model.SyncSchedule{
		{
			Type:            model.SyncTypePlayers,
			Enabled:         settings.PlayersEnabled,
			IntervalMinutes: playersInterval,
			NextRunAt:       now.Add(time.Duration(playersInterval) * time.Minute),
		},
		{
			Type:            model.SyncTypeStatistics,
			Enabled:         settings.StatisticsEnabled,
			IntervalMinutes: statisticsInterval,
			NextRunAt:       now.Add(time.Duration(statisticsInterval) * time.Minute),
		},
	}
}

// intervalMinutes traduce la configuración a minutos y nunca deja un valor por
// debajo del mínimo, para que un settings incompleto no viole el CHECK de la DB.
func intervalMinutes(d time.Duration) int {
	minutes := int(d / time.Minute)
	if minutes < model.MinSyncIntervalMinutes {
		return model.MinSyncIntervalMinutes
	}
	return minutes
}
