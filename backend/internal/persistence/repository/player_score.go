package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerScoreDAO interface {
	Insert(ctx context.Context, tx *sql.Tx, scores []model.PlayerScore) error
	ExistsFor(ctx context.Context, playerID int64, strategyID int, statisticsID int64) (bool, error)
	ListByPlayer(ctx context.Context, playerID int64) ([]model.PlayerScore, error)
}

type PlayerScoreRepositoryImpl struct {
	db  *sql.DB
	dao PlayerScoreDAO
}

func NewPlayerScoreRepository(db *sql.DB, dao PlayerScoreDAO) *PlayerScoreRepositoryImpl {
	return &PlayerScoreRepositoryImpl{db: db, dao: dao}
}

func (r *PlayerScoreRepositoryImpl) Save(ctx context.Context, scores []model.PlayerScore) error {
	if len(scores) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.dao.Insert(ctx, tx, scores); err != nil {
		return fmt.Errorf("insert player scores: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PlayerScoreRepositoryImpl) HasScoreFor(ctx context.Context, playerID int64, strategyID int, statisticsID int64) (bool, error) {
	return r.dao.ExistsFor(ctx, playerID, strategyID, statisticsID)
}

func (r *PlayerScoreRepositoryImpl) History(ctx context.Context, playerID int64) ([]model.PlayerScore, error) {
	scores, err := r.dao.ListByPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("repository list player scores: %w", err)
	}
	return scores, nil
}
