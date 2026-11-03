package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type SyncRunDAO interface {
	InsertRunning(ctx context.Context, tx *sql.Tx, run model.SyncRun) (int64, bool, error)
	InsertSkipped(ctx context.Context, tx *sql.Tx, run model.SyncRun) (int64, error)
	Finish(ctx context.Context, id int64, result model.SyncRunResult) error
	Get(ctx context.Context, id int64) (model.SyncRun, bool, error)
	List(ctx context.Context, typ *model.SyncType, offset, limit int) ([]model.SyncRun, int, error)
	MarkInterrupted(ctx context.Context) (int64, error)
}

type SyncRunRepositoryImpl struct {
	db  *sql.DB
	dao SyncRunDAO
}

func NewSyncRunRepository(db *sql.DB, dao SyncRunDAO) *SyncRunRepositoryImpl {
	return &SyncRunRepositoryImpl{db: db, dao: dao}
}

// Start crea el run como running o, si ya hay uno del mismo tipo, como skipped.
// La transacción garantiza que no quede un running sin su fila correspondiente.
func (r *SyncRunRepositoryImpl) Start(ctx context.Context, typ model.SyncType, trigger model.SyncTrigger, actorID *int64) (model.SyncRun, error) {
	if r.db == nil {
		return model.SyncRun{}, fmt.Errorf("database connection is nil")
	}

	now := time.Now()
	run := model.SyncRun{
		Type:        typ,
		Trigger:     trigger,
		Status:      model.SyncStatusRunning,
		StartedAt:   now,
		TriggeredBy: actorID,
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.SyncRun{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	id, started, err := r.dao.InsertRunning(ctx, tx, run)
	if err != nil {
		return model.SyncRun{}, fmt.Errorf("insert running sync run: %w", err)
	}

	if started {
		run.ID = id
	} else {
		skipped := run
		skipped.Status = model.SyncStatusSkipped
		skipped.FinishedAt = &now
		skipped.Message = "ya hay una sincronización de este tipo en curso"

		id, err = r.dao.InsertSkipped(ctx, tx, skipped)
		if err != nil {
			return model.SyncRun{}, fmt.Errorf("insert skipped sync run: %w", err)
		}
		skipped.ID = id
		run = skipped
	}

	if err := tx.Commit(); err != nil {
		return model.SyncRun{}, fmt.Errorf("commit transaction: %w", err)
	}

	return run, nil
}

func (r *SyncRunRepositoryImpl) Finish(ctx context.Context, id int64, result model.SyncRunResult) error {
	if err := r.dao.Finish(ctx, id, result); err != nil {
		return fmt.Errorf("repository finish sync run: %w", err)
	}
	return nil
}

func (r *SyncRunRepositoryImpl) Get(ctx context.Context, id int64) (model.SyncRun, bool, error) {
	return r.dao.Get(ctx, id)
}

func (r *SyncRunRepositoryImpl) List(ctx context.Context, typ *model.SyncType, offset, limit int) ([]model.SyncRun, int, error) {
	return r.dao.List(ctx, typ, offset, limit)
}

func (r *SyncRunRepositoryImpl) MarkInterrupted(ctx context.Context) (int64, error) {
	return r.dao.MarkInterrupted(ctx)
}
