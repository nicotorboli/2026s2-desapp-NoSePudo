package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type AccountSql struct {
	Db *sql.DB
}

func NewAccountDao(db *sql.DB) *AccountSql {
	return &AccountSql{Db: db}
}

func (dao *AccountSql) Insert(ctx context.Context, tx *sql.Tx, account model.Account) error {
	const query = `
		INSERT INTO user_accounts (user_id, available_balance, reserved_balance)
		VALUES ($1, $2, $3)`

	_, err := execOn(ctx, tx, dao.Db, query, account.UserID, int64(account.AvailableBalance), int64(account.ReservedBalance))
	if err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == uniqueViolation {
			return model.ErrAccountAlreadyExists
		}
		return fmt.Errorf("insertar la cuenta de mercado: %w", err)
	}

	return nil
}

func (dao *AccountSql) LockByUser(ctx context.Context, tx *sql.Tx, userID int64) (model.Account, error) {
	const query = `
		SELECT user_id, available_balance, reserved_balance
		FROM user_accounts
		WHERE user_id = $1
		FOR UPDATE`

	if tx == nil {
		return model.Account{}, errLockNeedsTransaction
	}

	return scanAccount(tx.QueryRowContext(ctx, query, userID))
}

func (dao *AccountSql) GetByUser(ctx context.Context, userID int64) (model.Account, error) {
	const query = `
		SELECT user_id, available_balance, reserved_balance
		FROM user_accounts
		WHERE user_id = $1`

	if dao.Db == nil {
		return model.Account{}, errNilDatabase
	}

	return scanAccount(dao.Db.QueryRowContext(ctx, query, userID))
}

func (dao *AccountSql) UpdateBalances(ctx context.Context, tx *sql.Tx, account model.Account) error {
	const query = `
		UPDATE user_accounts
		SET available_balance = $2, reserved_balance = $3, updated_at = NOW()
		WHERE user_id = $1`

	result, err := execOn(ctx, tx, dao.Db, query, account.UserID, int64(account.AvailableBalance), int64(account.ReservedBalance))
	if err != nil {
		return fmt.Errorf("actualizar los saldos: %w", err)
	}

	return requireOneRow(result, model.ErrAccountNotFound)
}

func scanAccount(row *sql.Row) (model.Account, error) {
	var (
		account   model.Account
		available int64
		reserved  int64
	)

	if err := row.Scan(&account.UserID, &available, &reserved); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Account{}, model.ErrAccountNotFound
		}
		return model.Account{}, fmt.Errorf("leer la cuenta de mercado: %w", err)
	}

	account.AvailableBalance = model.Credits(available)
	account.ReservedBalance = model.Credits(reserved)

	return account, nil
}
