package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type LedgerSql struct {
	Db *sql.DB
}

func NewLedgerDao(db *sql.DB) *LedgerSql {
	return &LedgerSql{Db: db}
}

func (dao *LedgerSql) NextSettlementGroup(ctx context.Context, tx *sql.Tx) (int64, error) {
	const query = `SELECT nextval('ledger_settlement_seq')`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return 0, err
	}

	var settlementGroup int64
	if err := querier.QueryRowContext(ctx, query).Scan(&settlementGroup); err != nil {
		return 0, fmt.Errorf("pedir el grupo de liquidación: %w", err)
	}

	return settlementGroup, nil
}

func (dao *LedgerSql) Insert(ctx context.Context, tx *sql.Tx, entry model.LedgerEntry) (model.LedgerEntry, error) {
	const query = `
		INSERT INTO ledger_entries (settlement_group, trade_id, user_id, reason, amount, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.LedgerEntry{}, err
	}

	err = querier.QueryRowContext(ctx, query,
		entry.SettlementGroup, entry.TradeID, entry.UserID,
		string(entry.Reason), int64(entry.Amount), entry.Description,
	).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return model.LedgerEntry{}, fmt.Errorf("insertar el asiento: %w", err)
	}

	return entry, nil
}

func (dao *LedgerSql) UnbalancedSettlementGroups(ctx context.Context) ([]int64, error) {
	const query = `
		SELECT settlement_group
		FROM ledger_entries
		GROUP BY settlement_group
		HAVING SUM(amount) <> 0`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("buscar los grupos de liquidación desbalanceados: %w", err)
	}
	defer func() { _ = rows.Close() }()

	groups := make([]int64, 0)
	for rows.Next() {
		var group int64
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("leer el grupo de liquidación: %w", err)
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer los grupos de liquidación: %w", err)
	}

	return groups, nil
}
