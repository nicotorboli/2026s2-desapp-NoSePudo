package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type LeagueCoefficientDAO interface {
	Insert(ctx context.Context, coefficient model.LeagueCoefficient) error
	CurrentFor(ctx context.Context, leagueCode string) (model.LeagueCoefficient, bool, error)
	AtOrBefore(ctx context.Context, leagueCode string, at time.Time) (model.LeagueCoefficient, bool, error)
}

type LeagueCoefficientRepositoryImpl struct {
	dao LeagueCoefficientDAO
}

func NewLeagueCoefficientRepository(dao LeagueCoefficientDAO) *LeagueCoefficientRepositoryImpl {
	return &LeagueCoefficientRepositoryImpl{dao: dao}
}

func (r *LeagueCoefficientRepositoryImpl) Set(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error {
	if err := r.dao.Insert(ctx, model.LeagueCoefficient{LeagueCode: leagueCode, Coefficient: coefficient, SetBy: actorID}); err != nil {
		return fmt.Errorf("repository set league coefficient: %w", err)
	}
	return nil
}

func (r *LeagueCoefficientRepositoryImpl) Current(ctx context.Context, leagueCode string) (model.LeagueCoefficient, bool, error) {
	return r.dao.CurrentFor(ctx, leagueCode)
}

func (r *LeagueCoefficientRepositoryImpl) AtOrBefore(ctx context.Context, leagueCode string, at time.Time) (model.LeagueCoefficient, bool, error) {
	return r.dao.AtOrBefore(ctx, leagueCode, at)
}
