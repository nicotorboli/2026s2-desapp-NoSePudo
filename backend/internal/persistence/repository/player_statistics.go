package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type PlayerStatisticsRepository interface {
	SaveStatistics(ctx context.Context, rows []model.PlayerStatistics) (int, int, error)
}

type PlayerStatisticsRepositoryImpl struct {
	db  *sql.DB
	dao dao.PlayerStatisticsDAO
}

func NewPlayerStatisticsRepository(db *sql.DB, d dao.PlayerStatisticsDAO) *PlayerStatisticsRepositoryImpl {
	return &PlayerStatisticsRepositoryImpl{
		db:  db,
		dao: d,
	}
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
