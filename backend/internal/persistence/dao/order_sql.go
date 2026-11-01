package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const orderColumns = `id, user_id, player_id, side, time_in_force, status, price,
	quantity, executed_quantity, market_maker, created_at, updated_at`

type OrderSql struct {
	Db *sql.DB
}

func NewOrderDao(db *sql.DB) *OrderSql {
	return &OrderSql{Db: db}
}

func (dao *OrderSql) Insert(ctx context.Context, tx *sql.Tx, order model.Order) (model.Order, error) {
	const query = `
		INSERT INTO market_orders
			(user_id, player_id, side, time_in_force, status, price, quantity, executed_quantity, market_maker)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.Order{}, err
	}

	err = querier.QueryRowContext(ctx, query,
		order.UserID, order.PlayerID, string(order.Side), string(order.TimeInForce), string(order.Status),
		int64(order.Price), order.Quantity, order.ExecutedQuantity, order.MarketMaker,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return model.Order{}, fmt.Errorf("insertar la orden: %w", err)
	}

	return order, nil
}

func (dao *OrderSql) GetByID(ctx context.Context, tx *sql.Tx, id int64) (model.Order, error) {
	const query = `SELECT ` + orderColumns + ` FROM market_orders WHERE id = $1`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.Order{}, err
	}

	return scanOrder(querier.QueryRowContext(ctx, query, id))
}

func (dao *OrderSql) Update(ctx context.Context, tx *sql.Tx, order model.Order) error {
	const query = `
		UPDATE market_orders
		SET executed_quantity = $2, status = $3, updated_at = NOW()
		WHERE id = $1`

	result, err := execOn(ctx, tx, dao.Db, query, order.ID, order.ExecutedQuantity, string(order.Status))
	if err != nil {
		return fmt.Errorf("actualizar la orden: %w", err)
	}

	return requireOneRow(result, model.ErrOrderNotFound)
}

func (dao *OrderSql) LockLiveBook(ctx context.Context, tx *sql.Tx, playerID int64, side model.OrderSide) ([]model.Order, error) {
	const query = `
		SELECT ` + orderColumns + `
		FROM market_orders
		WHERE player_id = $1 AND side = $2 AND status IN ('PENDING', 'PARTIALLY_FILLED')
		ORDER BY price, created_at, id
		FOR UPDATE`

	if tx == nil {
		return nil, errLockNeedsTransaction
	}

	rows, err := tx.QueryContext(ctx, query, playerID, string(side))
	if err != nil {
		return nil, fmt.Errorf("leer el libro: %w", err)
	}

	return collectOrders(rows)
}

func (dao *OrderSql) ListLiveByUser(ctx context.Context, userID int64, playerID *int64) ([]model.Order, error) {
	const query = `
		SELECT ` + orderColumns + `
		FROM market_orders
		WHERE user_id = $1
		  AND status IN ('PENDING', 'PARTIALLY_FILLED')
		  AND ($2::BIGINT IS NULL OR player_id = $2)
		ORDER BY created_at DESC, id DESC`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, userID, playerID)
	if err != nil {
		return nil, fmt.Errorf("listar las órdenes vivas: %w", err)
	}

	return collectOrders(rows)
}

func (dao *OrderSql) ListLiveByPlayer(ctx context.Context, playerID int64) ([]model.Order, error) {
	const query = `
		SELECT ` + orderColumns + `
		FROM market_orders
		WHERE player_id = $1
		  AND status IN ('PENDING', 'PARTIALLY_FILLED')
		ORDER BY price ASC, created_at ASC, id ASC`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, fmt.Errorf("listar las órdenes del libro de mercado: %w", err)
	}

	return collectOrders(rows)
}

func (dao *OrderSql) GetLiveMarketMaker(ctx context.Context, tx *sql.Tx, playerID int64, side model.OrderSide) (model.Order, error) {
	const query = `
		SELECT ` + orderColumns + `
		FROM market_orders
		WHERE player_id = $1 AND side = $2 AND market_maker
		  AND status IN ('PENDING', 'PARTIALLY_FILLED')`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.Order{}, err
	}

	return scanOrder(querier.QueryRowContext(ctx, query, playerID, string(side)))
}

func collectOrders(rows *sql.Rows) ([]model.Order, error) {
	defer func() { _ = rows.Close() }()

	orders := make([]model.Order, 0)
	for rows.Next() {
		order, err := scanOrderFromRows(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las órdenes: %w", err)
	}

	return orders, nil
}

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrderInto(scanner orderScanner) (model.Order, error) {
	var (
		order       model.Order
		side        string
		timeInForce string
		status      string
		price       int64
	)

	err := scanner.Scan(
		&order.ID,
		&order.UserID,
		&order.PlayerID,
		&side,
		&timeInForce,
		&status,
		&price,
		&order.Quantity,
		&order.ExecutedQuantity,
		&order.MarketMaker,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return model.Order{}, err
	}

	order.Side = model.OrderSide(side)
	order.TimeInForce = model.TimeInForce(timeInForce)
	order.Status = model.OrderStatus(status)
	order.Price = model.Credits(price)

	return order, nil
}

func scanOrder(row *sql.Row) (model.Order, error) {
	order, err := scanOrderInto(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Order{}, model.ErrOrderNotFound
		}
		return model.Order{}, fmt.Errorf("leer la orden: %w", err)
	}

	return order, nil
}

func scanOrderFromRows(rows *sql.Rows) (model.Order, error) {
	order, err := scanOrderInto(rows)
	if err != nil {
		return model.Order{}, fmt.Errorf("leer la orden: %w", err)
	}

	return order, nil
}
