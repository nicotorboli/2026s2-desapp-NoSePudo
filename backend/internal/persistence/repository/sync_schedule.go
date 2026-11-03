package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type SyncScheduleDAO interface {
	List(ctx context.Context) ([]model.SyncSchedule, error)
	Get(ctx context.Context, typ model.SyncType) (model.SyncSchedule, bool, error)
	Ensure(ctx context.Context, schedule model.SyncSchedule) error
	Set(ctx context.Context, schedule model.SyncSchedule) (model.SyncSchedule, bool, error)
	EnableLegacyDefaults(ctx context.Context, now time.Time) (int64, error)
	ReserveDue(ctx context.Context, now time.Time) ([]model.SyncSchedule, error)
}

type SyncScheduleRepositoryImpl struct {
	dao SyncScheduleDAO
}

func NewSyncScheduleRepository(dao SyncScheduleDAO) *SyncScheduleRepositoryImpl {
	return &SyncScheduleRepositoryImpl{dao: dao}
}

func (r *SyncScheduleRepositoryImpl) List(ctx context.Context) ([]model.SyncSchedule, error) {
	schedules, err := r.dao.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository list sync schedules: %w", err)
	}
	return schedules, nil
}

func (r *SyncScheduleRepositoryImpl) Get(ctx context.Context, typ model.SyncType) (model.SyncSchedule, bool, error) {
	return r.dao.Get(ctx, typ)
}

func (r *SyncScheduleRepositoryImpl) Ensure(ctx context.Context, schedule model.SyncSchedule) error {
	if err := r.dao.Ensure(ctx, schedule); err != nil {
		return fmt.Errorf("repository ensure sync schedule: %w", err)
	}
	return nil
}

func (r *SyncScheduleRepositoryImpl) Set(ctx context.Context, schedule model.SyncSchedule) (model.SyncSchedule, bool, error) {
	return r.dao.Set(ctx, schedule)
}

func (r *SyncScheduleRepositoryImpl) EnableLegacyDefaults(ctx context.Context, now time.Time) (int64, error) {
	count, err := r.dao.EnableLegacyDefaults(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("repository enable legacy sync schedules: %w", err)
	}
	return count, nil
}

func (r *SyncScheduleRepositoryImpl) ReserveDue(ctx context.Context, now time.Time) ([]model.SyncSchedule, error) {
	due, err := r.dao.ReserveDue(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("repository reserve due sync schedules: %w", err)
	}
	return due, nil
}
