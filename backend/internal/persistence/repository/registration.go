package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type RegistrationRepository struct {
	db             *sql.DB
	userDao        *dao.UserSql
	accountDao     *dao.AccountSql
	ledgerDao      *dao.LedgerSql
	initialBalance model.Credits
}

func NewRegistrationRepository(
	db *sql.DB,
	userDao *dao.UserSql,
	accountDao *dao.AccountSql,
	ledgerDao *dao.LedgerSql,
	initialBalance model.Credits,
) *RegistrationRepository {
	return &RegistrationRepository{
		db:             db,
		userDao:        userDao,
		accountDao:     accountDao,
		ledgerDao:      ledgerDao,
		initialBalance: initialBalance,
	}
}

func (r *RegistrationRepository) InsertUserWithAccount(ctx context.Context, user model.User) (model.User, error) {
	if r.db == nil {
		return model.User{}, fmt.Errorf("database connection is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.User{}, fmt.Errorf("abrir la transacción del alta: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stored, err := r.userDao.InsertTx(ctx, tx, user)
	if err != nil {
		return model.User{}, err
	}

	if err = r.openAccount(ctx, tx, stored.ID); err != nil {
		return model.User{}, err
	}

	if err = tx.Commit(); err != nil {
		return model.User{}, fmt.Errorf("confirmar la transacción del alta: %w", err)
	}

	return stored, nil
}

func (r *RegistrationRepository) openAccount(ctx context.Context, tx *sql.Tx, userID int64) error {
	account := model.Account{UserID: userID, AvailableBalance: r.initialBalance}
	if err := r.accountDao.Insert(ctx, tx, account); err != nil {
		return fmt.Errorf("crear la cuenta del usuario %d: %w", userID, err)
	}

	if r.initialBalance == 0 {
		return nil
	}

	return r.grantInitialBalance(ctx, tx, userID)
}

func (r *RegistrationRepository) grantInitialBalance(ctx context.Context, tx *sql.Tx, userID int64) error {
	settlementGroup, err := r.ledgerDao.NextSettlementGroup(ctx, tx)
	if err != nil {
		return err
	}

	granted := userID
	entries := []model.LedgerEntry{
		{
			UserID:          &granted,
			Reason:          model.ReasonInitialGrant,
			Description:     "crédito inicial de la plataforma",
			SettlementGroup: settlementGroup,
			Amount:          r.initialBalance,
		},
		{
			Reason:          model.ReasonInitialGrant,
			Description:     "contrapartida del crédito inicial",
			SettlementGroup: settlementGroup,
			Amount:          -r.initialBalance,
		},
	}

	for i := range entries {
		if _, insertErr := r.ledgerDao.Insert(ctx, tx, entries[i]); insertErr != nil {
			return insertErr
		}
	}

	return nil
}
