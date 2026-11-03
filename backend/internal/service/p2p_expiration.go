package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type P2PExpirationRepository interface {
	ExpirePending(ctx context.Context, now time.Time) error
}

type P2PExpirationJob struct {
	repo    P2PExpirationRepository
	logger  *slog.Logger
	now     func() time.Time
	cancel  context.CancelFunc
	done    chan struct{}
	tick    time.Duration
	mu      sync.Mutex
	started bool
}

func NewP2PExpirationJob(repo P2PExpirationRepository, now func() time.Time, logger *slog.Logger) *P2PExpirationJob {
	if now == nil {
		now = time.Now
	}
	return &P2PExpirationJob{
		repo:   repo,
		now:    now,
		logger: logger,
		tick:   30 * time.Second,
		done:   make(chan struct{}),
	}
}

func (j *P2PExpirationJob) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	j.mu.Lock()
	j.cancel = cancel
	j.started = true
	j.mu.Unlock()

	j.tickOnce(ctx)
	go j.loop(ctx)
}

func (j *P2PExpirationJob) Stop() {
	j.mu.Lock()
	cancel := j.cancel
	started := j.started
	j.mu.Unlock()

	if !started {
		return
	}
	if cancel != nil {
		cancel()
	}

	select {
	case <-j.done:
	case <-time.After(5 * time.Second):
	}
}

func (j *P2PExpirationJob) loop(ctx context.Context) {
	defer close(j.done)

	ticker := time.NewTicker(j.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.tickOnce(ctx)
		}
	}
}

func (j *P2PExpirationJob) tickOnce(ctx context.Context) {
	if err := j.repo.ExpirePending(ctx, j.now()); err != nil {
		if j.logger != nil {
			j.logger.ErrorContext(ctx, "failed to expire pending p2p sessions", "error", err.Error())
		}
	}
}
