package repository

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoringStrategyDAO interface {
	List(ctx context.Context) ([]model.ScoringStrategy, error)
}

type ScoringStrategyRepositoryImpl struct {
	dao ScoringStrategyDAO
}

func NewScoringStrategyRepository(dao ScoringStrategyDAO) *ScoringStrategyRepositoryImpl {
	return &ScoringStrategyRepositoryImpl{dao: dao}
}

func (r *ScoringStrategyRepositoryImpl) List(ctx context.Context) ([]model.ScoringStrategy, error) {
	return r.dao.List(ctx)
}
