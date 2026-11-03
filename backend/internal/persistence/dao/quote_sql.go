package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type QuoteSql struct {
	Db *sql.DB
}

func NewQuoteDao(db *sql.DB) *QuoteSql {
	return &QuoteSql{Db: db}
}

func (dao *QuoteSql) InsertInitialIfAbsent(ctx context.Context, tx *sql.Tx, playerID int64, price model.Credits) (bool, error) {
	const query = `
		INSERT INTO player_quotes (player_id, price, strategy, strategy_version)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (player_id) WHERE strategy = 'initial' DO NOTHING`

	result, err := execOn(ctx, tx, dao.Db, query,
		playerID, int64(price), model.InitialQuoteStrategy, model.InitialQuoteStrategyVersion,
	)
	if err != nil {
		return false, fmt.Errorf("insertar la cotización inicial: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("contar las filas afectadas: %w", err)
	}

	return affected == 1, nil
}

func (dao *QuoteSql) Insert(ctx context.Context, tx *sql.Tx, quote model.PlayerQuote) (model.PlayerQuote, error) {
	const query = `
		INSERT INTO player_quotes (player_id, price, strategy, strategy_version, score_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.PlayerQuote{}, err
	}

	err = querier.QueryRowContext(ctx, query,
		quote.PlayerID, int64(quote.Price), quote.Strategy, quote.StrategyVersion, quote.ScoreID,
	).Scan(&quote.ID, &quote.CreatedAt)
	if err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == uniqueViolation {
			return model.PlayerQuote{}, model.ErrQuoteAlreadyExists
		}
		return model.PlayerQuote{}, fmt.Errorf("insertar la cotización: %w", err)
	}

	return quote, nil
}

func (dao *QuoteSql) GetLatestByPlayer(ctx context.Context, tx *sql.Tx, playerID int64) (model.PlayerQuote, error) {
	const query = `
		SELECT id, player_id, price, strategy, strategy_version, created_at
		FROM player_quotes
		WHERE player_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.PlayerQuote{}, err
	}

	var (
		quote model.PlayerQuote
		price int64
	)

	err = querier.QueryRowContext(ctx, query, playerID).Scan(
		&quote.ID,
		&quote.PlayerID,
		&price,
		&quote.Strategy,
		&quote.StrategyVersion,
		&quote.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PlayerQuote{}, model.ErrQuoteNotFound
		}
		return model.PlayerQuote{}, fmt.Errorf("leer la cotización vigente: %w", err)
	}

	quote.Price = model.Credits(price)

	return quote, nil
}

func (dao *QuoteSql) LatestPriceForPlayers(ctx context.Context, playerIDs []int64) (map[int64]model.Credits, error) {
	const query = `
		SELECT DISTINCT ON (player_id) player_id, price
		FROM player_quotes
		WHERE player_id = ANY($1)
		ORDER BY player_id, created_at DESC, id DESC`

	prices := make(map[int64]model.Credits, len(playerIDs))
	if len(playerIDs) == 0 {
		return prices, nil
	}
	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, pq.Array(playerIDs))
	if err != nil {
		return nil, fmt.Errorf("leer las cotizaciones vigentes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			playerID int64
			price    int64
		)
		if err := rows.Scan(&playerID, &price); err != nil {
			return nil, fmt.Errorf("leer la cotización vigente: %w", err)
		}
		prices[playerID] = model.Credits(price)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las cotizaciones vigentes: %w", err)
	}

	return prices, nil
}

func (dao *QuoteSql) ListByPlayer(ctx context.Context, playerID int64) ([]model.PlayerQuote, error) {
	const query = `
		SELECT id, player_id, price, strategy, strategy_version, score_id, created_at
		FROM player_quotes
		WHERE player_id = $1
		ORDER BY created_at DESC, id DESC`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, fmt.Errorf("leer el historial de cotizaciones: %w", err)
	}
	defer func() { _ = rows.Close() }()

	quotes := make([]model.PlayerQuote, 0)
	for rows.Next() {
		var (
			quote   model.PlayerQuote
			price   int64
			scoreID sql.NullInt64
		)
		if err := rows.Scan(
			&quote.ID,
			&quote.PlayerID,
			&price,
			&quote.Strategy,
			&quote.StrategyVersion,
			&scoreID,
			&quote.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("leer una cotización del historial: %w", err)
		}
		quote.Price = model.Credits(price)
		if scoreID.Valid {
			value := scoreID.Int64
			quote.ScoreID = &value
		}
		quotes = append(quotes, quote)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer el historial de cotizaciones: %w", err)
	}

	return quotes, nil
}
