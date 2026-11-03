package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type TradeSql struct {
	Db *sql.DB
}

func NewTradeDao(db *sql.DB) *TradeSql {
	return &TradeSql{Db: db}
}

func (dao *TradeSql) Insert(ctx context.Context, tx *sql.Tx, trade model.Trade) (model.Trade, error) {
	const query = `
		INSERT INTO market_trades
			(buy_order_id, sell_order_id, buyer_id, seller_id, player_id, price, quantity, total_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, executed_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.Trade{}, err
	}

	err = querier.QueryRowContext(ctx, query,
		trade.BuyOrderID, trade.SellOrderID, trade.BuyerID, trade.SellerID, trade.PlayerID,
		int64(trade.Price), trade.Quantity, int64(trade.TotalAmount),
	).Scan(&trade.ID, &trade.ExecutedAt)
	if err != nil {
		return model.Trade{}, fmt.Errorf("insertar la operación: %w", err)
	}

	return trade, nil
}

func (dao *TradeSql) CountByParticipant(ctx context.Context, userID int64) (int64, error) {
	const query = `SELECT COUNT(*) FROM market_trades WHERE buyer_id = $1 OR seller_id = $1`

	if dao.Db == nil {
		return 0, errNilDatabase
	}

	var total int64
	if err := dao.Db.QueryRowContext(ctx, query, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("contar las operaciones: %w", err)
	}

	return total, nil
}

func (dao *TradeSql) ListByParticipant(ctx context.Context, userID int64, limit, offset int) ([]model.Trade, error) {
	const query = `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id, player_id,
		       price, quantity, total_amount, executed_at
		FROM market_trades
		WHERE buyer_id = $1 OR seller_id = $1
		ORDER BY executed_at DESC, id DESC
		LIMIT $2 OFFSET $3`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listar las operaciones: %w", err)
	}
	defer func() { _ = rows.Close() }()

	trades := make([]model.Trade, 0, limit)
	for rows.Next() {
		var (
			trade       model.Trade
			price       int64
			totalAmount int64
		)

		err := rows.Scan(
			&trade.ID,
			&trade.BuyOrderID,
			&trade.SellOrderID,
			&trade.BuyerID,
			&trade.SellerID,
			&trade.PlayerID,
			&price,
			&trade.Quantity,
			&totalAmount,
			&trade.ExecutedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("leer la operación: %w", err)
		}

		trade.Price = model.Credits(price)
		trade.TotalAmount = model.Credits(totalAmount)
		trades = append(trades, trade)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las operaciones: %w", err)
	}

	return trades, nil
}
