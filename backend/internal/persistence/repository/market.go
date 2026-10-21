package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type MarketOperation func(ctx context.Context, session *MarketSession) error

type MarketRepository struct {
	db          *sql.DB
	playerDao   *dao.PlayerSql
	accountDao  *dao.AccountSql
	positionDao *dao.PositionSql
	orderDao    *dao.OrderSql
	quoteDao    *dao.QuoteSql
}

func NewMarketRepository(
	db *sql.DB,
	playerDao *dao.PlayerSql,
	accountDao *dao.AccountSql,
	positionDao *dao.PositionSql,
	orderDao *dao.OrderSql,
	quoteDao *dao.QuoteSql,
) *MarketRepository {
	return &MarketRepository{
		db:          db,
		playerDao:   playerDao,
		accountDao:  accountDao,
		positionDao: positionDao,
		orderDao:    orderDao,
		quoteDao:    quoteDao,
	}
}

func (r *MarketRepository) WithPlayerLocked(ctx context.Context, playerID int64, operation MarketOperation) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir la transacción del mercado: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if lockErr := r.playerDao.LockPlayer(ctx, tx, playerID); lockErr != nil {
		if errors.Is(lockErr, dao.ErrNotFound) {
			return ErrNotFound
		}
		return lockErr
	}

	session := &MarketSession{
		tx:          tx,
		accountDao:  r.accountDao,
		positionDao: r.positionDao,
		orderDao:    r.orderDao,
		quoteDao:    r.quoteDao,
	}

	if operationErr := operation(ctx, session); operationErr != nil {
		return operationErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("confirmar la transacción del mercado: %w", commitErr)
	}

	return nil
}

type MarketSession struct {
	tx          *sql.Tx
	accountDao  *dao.AccountSql
	positionDao *dao.PositionSql
	orderDao    *dao.OrderSql
	quoteDao    *dao.QuoteSql
}

func (s *MarketSession) LockAccount(ctx context.Context, userID int64) (model.Account, error) {
	return s.accountDao.LockByUser(ctx, s.tx, userID)
}

func (s *MarketSession) LockPosition(ctx context.Context, userID, playerID int64) (model.Position, error) {
	return s.positionDao.LockByUserAndPlayer(ctx, s.tx, userID, playerID)
}

func (s *MarketSession) LockOppositeBook(ctx context.Context, playerID int64, incomingSide model.OrderSide) ([]model.Order, error) {
	return s.orderDao.LockLiveBook(ctx, s.tx, playerID, model.OppositeSide(incomingSide))
}

func (s *MarketSession) LatestQuote(ctx context.Context, playerID int64) (model.PlayerQuote, error) {
	return s.quoteDao.GetLatestByPlayer(ctx, s.tx, playerID)
}
