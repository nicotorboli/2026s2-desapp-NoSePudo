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
			(buy_order_id, sell_order_id, buyer_id, seller_id, player_id, price, quantity, total_amount, p2p_session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, executed_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.Trade{}, err
	}

	var (
		buyOrderID  any = trade.BuyOrderID
		sellOrderID any = trade.SellOrderID
	)
	if trade.BuyOrderID == 0 {
		buyOrderID = nil
	}
	if trade.SellOrderID == 0 {
		sellOrderID = nil
	}

	err = querier.QueryRowContext(ctx, query,
		buyOrderID, sellOrderID, trade.BuyerID, trade.SellerID, trade.PlayerID,
		int64(trade.Price), trade.Quantity, int64(trade.TotalAmount), trade.P2PSessionID,
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
		       price, quantity, total_amount, executed_at, p2p_session_id
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
			trade        model.Trade
			price        int64
			totalAmount  int64
			buyOrderID   sql.NullInt64
			sellOrderID  sql.NullInt64
			p2pSessionID sql.NullInt64
		)

		err := rows.Scan(
			&trade.ID,
			&buyOrderID,
			&sellOrderID,
			&trade.BuyerID,
			&trade.SellerID,
			&trade.PlayerID,
			&price,
			&trade.Quantity,
			&totalAmount,
			&trade.ExecutedAt,
			&p2pSessionID,
		)
		if err != nil {
			return nil, fmt.Errorf("leer la operación: %w", err)
		}

		if buyOrderID.Valid {
			trade.BuyOrderID = buyOrderID.Int64
		}
		if sellOrderID.Valid {
			trade.SellOrderID = sellOrderID.Int64
		}
		trade.Price = model.Credits(price)
		trade.TotalAmount = model.Credits(totalAmount)
		if p2pSessionID.Valid {
			id := p2pSessionID.Int64
			trade.P2PSessionID = &id
		}
		trades = append(trades, trade)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las operaciones: %w", err)
	}

	return trades, nil
}
