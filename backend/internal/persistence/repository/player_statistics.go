package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerStatisticsDAO interface {
	InsertStatistics(ctx context.Context, tx *sql.Tx, rows []model.PlayerStatistics) (int64, error)
	ListByPlayerID(ctx context.Context, playerID int64) ([]model.PlayerStatistics, error)
}

type PlayerStatisticsRepositoryImpl struct {
	db  *sql.DB
	dao PlayerStatisticsDAO
}

func NewPlayerStatisticsRepository(db *sql.DB, d PlayerStatisticsDAO) *PlayerStatisticsRepositoryImpl {
	return &PlayerStatisticsRepositoryImpl{
		db:  db,
		dao: d,
	}
}

func (r *PlayerStatisticsRepositoryImpl) GetPlayerStatistics(ctx context.Context, playerID int64) ([]model.PlayerStatistics, error) {
	stats, err := r.dao.ListByPlayerID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("repository list player statistics: %w", err)
	}
	return stats, nil
}

func (r *PlayerStatisticsRepositoryImpl) SaveStatistics(ctx context.Context, rows []model.PlayerStatistics) (int, int, error) {
	processed := len(rows)
	if processed == 0 {
		return 0, 0, nil
	}

	if r.db == nil {
		return 0, 0, fmt.Errorf("database connection is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	inserted, err := r.dao.InsertStatistics(ctx, tx, rows)
	if err != nil {
		return 0, 0, fmt.Errorf("insert statistics: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit transaction: %w", err)
	}

	return processed, int(inserted), nil
}
