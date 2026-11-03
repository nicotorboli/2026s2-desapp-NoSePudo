package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type SyncScheduleSql struct {
	db *sql.DB
}

func NewSyncScheduleDao(db *sql.DB) *SyncScheduleSql {
	return &SyncScheduleSql{db: db}
}

func (d *SyncScheduleSql) List(ctx context.Context) ([]model.SyncSchedule, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}

	const query = `
		SELECT type, enabled, interval_minutes, next_run_at, updated_at, updated_by
		FROM sync_schedules
		ORDER BY type`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list sync schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	schedules := make([]model.SyncSchedule, 0, 2)
	for rows.Next() {
		schedule, err := scanSyncSchedule(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan sync schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync schedules: %w", err)
	}

	return schedules, nil
}

func (d *SyncScheduleSql) Get(ctx context.Context, typ model.SyncType) (model.SyncSchedule, bool, error) {
	if d.db == nil {
		return model.SyncSchedule{}, false, errNilDatabase
	}

	const query = `
		SELECT type, enabled, interval_minutes, next_run_at, updated_at, updated_by
		FROM sync_schedules
		WHERE type = $1`

	schedule, err := scanSyncSchedule(d.db.QueryRowContext(ctx, query, string(typ)).Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.SyncSchedule{}, false, nil
		}
		return model.SyncSchedule{}, false, fmt.Errorf("get sync schedule: %w", err)
	}

	return schedule, true, nil
}

func (d *SyncScheduleSql) Ensure(ctx context.Context, schedule model.SyncSchedule) error {
	if d.db == nil {
		return errNilDatabase
	}

	const query = `
		INSERT INTO sync_schedules (type, enabled, interval_minutes, next_run_at, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (type) DO NOTHING`

	if _, err := d.db.ExecContext(
		ctx,
		query,
		string(schedule.Type),
		schedule.Enabled,
		schedule.IntervalMinutes,
		schedule.NextRunAt,
		nullableInt64(schedule.UpdatedBy),
	); err != nil {
		return fmt.Errorf("ensure sync schedule: %w", err)
	}

	return nil
}

func (d *SyncScheduleSql) Set(ctx context.Context, schedule model.SyncSchedule) (model.SyncSchedule, bool, error) {
	if d.db == nil {
		return model.SyncSchedule{}, false, errNilDatabase
	}

	const query = `
		UPDATE sync_schedules
		SET enabled = $1, interval_minutes = $2, next_run_at = $3, updated_at = NOW(), updated_by = $4
		WHERE type = $5
		RETURNING type, enabled, interval_minutes, next_run_at, updated_at, updated_by`

	updated, err := scanSyncSchedule(d.db.QueryRowContext(
		ctx,
		query,
		schedule.Enabled,
		schedule.IntervalMinutes,
		schedule.NextRunAt,
		nullableInt64(schedule.UpdatedBy),
		string(schedule.Type),
	).Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.SyncSchedule{}, false, nil
		}
		return model.SyncSchedule{}, false, fmt.Errorf("set sync schedule: %w", err)
	}

	return updated, true, nil
}

// ReserveDue reclama los schedules habilitados que ya vencieron y adelanta su
// próximo disparo. El UPDATE es atómico: si hubiera más de una instancia, solo
// una se queda con cada fila.
func (d *SyncScheduleSql) ReserveDue(ctx context.Context, now time.Time) ([]model.SyncSchedule, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}

	const query = `
		UPDATE sync_schedules
		SET next_run_at = $1 + (interval_minutes * INTERVAL '1 minute'), updated_at = NOW()
		WHERE enabled = TRUE AND next_run_at <= $1
		RETURNING type, enabled, interval_minutes, next_run_at, updated_at, updated_by`

	rows, err := d.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("reserve due sync schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	due := make([]model.SyncSchedule, 0, 2)
	for rows.Next() {
		schedule, err := scanSyncSchedule(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan due sync schedule: %w", err)
		}
		due = append(due, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due sync schedules: %w", err)
	}

	return due, nil
}

func scanSyncSchedule(scan func(dest ...any) error) (model.SyncSchedule, error) {
	var schedule model.SyncSchedule
	var updatedAt time.Time
	var nextRunAt time.Time
	var updatedBy sql.NullInt64

	if err := scan(
		&schedule.Type,
		&schedule.Enabled,
		&schedule.IntervalMinutes,
		&nextRunAt,
		&updatedAt,
		&updatedBy,
	); err != nil {
		return model.SyncSchedule{}, err
	}

	schedule.UpdatedAt = updatedAt
	schedule.NextRunAt = nextRunAt
	if updatedBy.Valid {
		schedule.UpdatedBy = &updatedBy.Int64
	}

	return schedule, nil
}
