package repository

import (
	"context"
	"math"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

const defaultTradePageSize = 20

type PortfolioRepository struct {
	accountDao  *dao.AccountSql
	positionDao *dao.PositionSql
	quoteDao    *dao.QuoteSql
	tradeDao    *dao.TradeSql
}

func NewPortfolioRepository(
	accountDao *dao.AccountSql,
	positionDao *dao.PositionSql,
	quoteDao *dao.QuoteSql,
	tradeDao *dao.TradeSql,
) *PortfolioRepository {
	return &PortfolioRepository{
		accountDao:  accountDao,
		positionDao: positionDao,
		quoteDao:    quoteDao,
		tradeDao:    tradeDao,
	}
}

func (r *PortfolioRepository) Account(ctx context.Context, userID int64) (model.Account, error) {
	return r.accountDao.GetByUser(ctx, userID)
}

func (r *PortfolioRepository) Positions(ctx context.Context, userID int64) ([]model.Position, error) {
	return r.positionDao.ListByUser(ctx, userID)
}

func (r *PortfolioRepository) LatestPrices(ctx context.Context, playerIDs []int64) (map[int64]model.Credits, error) {
	return r.quoteDao.LatestPriceForPlayers(ctx, playerIDs)
}

func (r *PortfolioRepository) Trades(ctx context.Context, userID int64, page, limit int) (model.PageResult[model.Trade], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultTradePageSize
	}

	total, err := r.tradeDao.CountByParticipant(ctx, userID)
	if err != nil {
		return model.PageResult[model.Trade]{}, err
	}

	trades, err := r.tradeDao.ListByParticipant(ctx, userID, limit, (page-1)*limit)
	if err != nil {
		return model.PageResult[model.Trade]{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return model.PageResult[model.Trade]{
		Items:      trades,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
