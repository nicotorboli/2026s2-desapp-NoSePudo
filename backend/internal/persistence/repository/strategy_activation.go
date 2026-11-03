package repository

import (
	"context"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type StrategyActivationDAO interface {
	Insert(ctx context.Context, activation model.StrategyActivation) error
	Latest(ctx context.Context) (model.StrategyActivation, bool, error)
}

type StrategyActivationRepositoryImpl struct {
	dao StrategyActivationDAO
}

func NewStrategyActivationRepository(dao StrategyActivationDAO) *StrategyActivationRepositoryImpl {
	return &StrategyActivationRepositoryImpl{dao: dao}
}

func (r *StrategyActivationRepositoryImpl) Activate(ctx context.Context, strategyID int, actorID int64) error {
	if err := r.dao.Insert(ctx, model.StrategyActivation{StrategyID: strategyID, ActivatedBy: actorID}); err != nil {
		return fmt.Errorf("repository activate strategy: %w", err)
	}
	return nil
}

func (r *StrategyActivationRepositoryImpl) Active(ctx context.Context) (model.StrategyActivation, bool, error) {
	return r.dao.Latest(ctx)
}
