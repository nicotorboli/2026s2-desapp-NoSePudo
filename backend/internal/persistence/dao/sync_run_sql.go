package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type SyncRunSql struct {
	db *sql.DB
}

func NewSyncRunDao(db *sql.DB) *SyncRunSql {
	return &SyncRunSql{db: db}
}

const syncRunColumns = `
	id, type, trigger, status, started_at, finished_at,
	total_processed, total_updated, total_deactivated,
	total_matched, total_unmatched, total_inserted,
	message, error, triggered_by`

// InsertRunning intenta crear el run en curso. Si ya hay uno del mismo tipo,
// el ON CONFLICT no inserta nada y devuelve started=false, sin error.
func (d *SyncRunSql) InsertRunning(ctx context.Context, tx *sql.Tx, run model.SyncRun) (int64, bool, error) {
	const query = `
		INSERT INTO sync_runs (type, trigger, status, started_at, triggered_by)
		VALUES ($1, $2, 'running', $3, $4)
		ON CONFLICT (type) WHERE status = 'running' DO NOTHING
		RETURNING id`

	var id int64
	if err := tx.QueryRowContext(
		ctx,
		query,
		string(run.Type),
		string(run.Trigger),
		run.StartedAt,
		nullableInt64(run.TriggeredBy),
	).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("insert running sync run: %w", err)
	}

	return id, true, nil
}

func (d *SyncRunSql) InsertSkipped(ctx context.Context, tx *sql.Tx, run model.SyncRun) (int64, error) {
	const query = `
		INSERT INTO sync_runs (type, trigger, status, started_at, finished_at, message)
		VALUES ($1, $2, 'skipped', $3, $4, $5)
		RETURNING id`

	var id int64
	if err := tx.QueryRowContext(
		ctx,
		query,
		string(run.Type),
		string(run.Trigger),
		run.StartedAt,
		run.FinishedAt,
		run.Message,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert skipped sync run: %w", err)
	}

	return id, nil
}

func (d *SyncRunSql) Finish(ctx context.Context, id int64, result model.SyncRunResult) error {
	if d.db == nil {
		return errNilDatabase
	}

	const query = `
		UPDATE sync_runs
		SET status = $2, finished_at = $3,
		    total_processed = $4, total_updated = $5, total_deactivated = $6,
		    total_matched = $7, total_unmatched = $8, total_inserted = $9,
		    message = $10, error = $11
		WHERE id = $1`

	if _, err := d.db.ExecContext(
		ctx,
		query,
		id,
		string(result.Status),
		result.FinishedAt,
		result.TotalProcessed,
		result.TotalUpdated,
		result.TotalDeactivated,
		result.TotalMatched,
		result.TotalUnmatched,
		result.TotalInserted,
		result.Message,
		result.Error,
	); err != nil {
		return fmt.Errorf("finish sync run: %w", err)
	}

	return nil
}

func (d *SyncRunSql) Get(ctx context.Context, id int64) (model.SyncRun, bool, error) {
	if d.db == nil {
		return model.SyncRun{}, false, errNilDatabase
	}

	const query = `SELECT ` + syncRunColumns + ` FROM sync_runs WHERE id = $1`
	// Idem al otro comment respecto al issue
	run, err := scanSyncRun(d.db.QueryRowContext(ctx, query, id).Scan) // NOSONAR
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.SyncRun{}, false, nil
		}
		return model.SyncRun{}, false, fmt.Errorf("get sync run: %w", err)
	}

	return run, true, nil
}

func (d *SyncRunSql) List(ctx context.Context, typ *model.SyncType, offset, limit int) ([]model.SyncRun, int, error) {
	if d.db == nil {
		return nil, 0, errNilDatabase
	}

	typeArg := ""
	if typ != nil {
		typeArg = string(*typ)
	}

	const query = `
		SELECT ` + syncRunColumns + `, COUNT(*) OVER() AS total
		FROM sync_runs
		WHERE ($1 = '' OR type = $1)
		ORDER BY started_at DESC, id DESC
		LIMIT $2 OFFSET $3`

	// En este caso omito el error porque no hay peligro de sql injection, solo esta usando un const dentro del mismo file
	rows, err := d.db.QueryContext(ctx, query, typeArg, limit, offset) // NOSONAR
	if err != nil {
		return nil, 0, fmt.Errorf("list sync runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	total := 0
	runs := make([]model.SyncRun, 0)
	for rows.Next() {
		var run model.SyncRun
		var finishedAt sql.NullTime
		var triggeredBy sql.NullInt64

		if err := rows.Scan(
			&run.ID,
			&run.Type,
			&run.Trigger,
			&run.Status,
			&run.StartedAt,
			&finishedAt,
			&run.TotalProcessed,
			&run.TotalUpdated,
			&run.TotalDeactivated,
			&run.TotalMatched,
			&run.TotalUnmatched,
			&run.TotalInserted,
			&run.Message,
			&run.Error,
			&triggeredBy,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan sync run: %w", err)
		}

		applyNullableRunFields(&run, finishedAt, triggeredBy)
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate sync runs: %w", err)
	}

	return runs, total, nil
}

func (d *SyncRunSql) MarkInterrupted(ctx context.Context) (int64, error) {
	if d.db == nil {
		return 0, errNilDatabase
	}

	const query = `
		UPDATE sync_runs
		SET status = 'failed', finished_at = NOW(), error = 'server restarted while the run was in progress'
		WHERE status = 'running'`

	result, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("mark interrupted sync runs: %w", err)
	}

	return result.RowsAffected()
}

func scanSyncRun(scan func(dest ...any) error) (model.SyncRun, error) {
	var run model.SyncRun
	var finishedAt sql.NullTime
	var triggeredBy sql.NullInt64

	if err := scan(
		&run.ID,
		&run.Type,
		&run.Trigger,
		&run.Status,
		&run.StartedAt,
		&finishedAt,
		&run.TotalProcessed,
		&run.TotalUpdated,
		&run.TotalDeactivated,
		&run.TotalMatched,
		&run.TotalUnmatched,
		&run.TotalInserted,
		&run.Message,
		&run.Error,
		&triggeredBy,
	); err != nil {
		return model.SyncRun{}, err
	}

	applyNullableRunFields(&run, finishedAt, triggeredBy)
	return run, nil
}

func applyNullableRunFields(run *model.SyncRun, finishedAt sql.NullTime, triggeredBy sql.NullInt64) {
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	if triggeredBy.Valid {
		run.TriggeredBy = &triggeredBy.Int64
	}
}

// nullableInt64 convierte un puntero en el valor que el driver espera para una
// columna nullable: nil cuando no hay actor, el valor cuando sí.
func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
