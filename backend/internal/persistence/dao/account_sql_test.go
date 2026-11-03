package dao_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestAccountSqlInsertYLeerLaCuenta(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)
	userID := seedUser(t, db)

	account := model.Account{UserID: userID, AvailableBalance: model.CreditsFromUnits(1000)}
	if err := accountDao.Insert(t.Context(), nil, account); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	stored, err := accountDao.GetByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("GetByUser devolvió %v", err)
	}
	if stored != account {
		t.Errorf("la cuenta leída es %+v, se esperaba %+v", stored, account)
	}
}

func TestAccountSqlGetByUserSinCuentaDevuelveElCentinela(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)

	_, err := accountDao.GetByUser(t.Context(), seedUser(t, db))
	if !errors.Is(err, model.ErrAccountNotFound) {
		t.Errorf("GetByUser devolvió %v, se esperaba ErrAccountNotFound", err)
	}
}

func TestAccountSqlInsertDuplicadoDevuelveElCentinela(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)
	userID := seedUser(t, db)

	account := model.Account{UserID: userID, AvailableBalance: model.CreditsFromUnits(1000)}
	if err := accountDao.Insert(t.Context(), nil, account); err != nil {
		t.Fatalf("el primer Insert devolvió %v", err)
	}

	if err := accountDao.Insert(t.Context(), nil, account); !errors.Is(err, model.ErrAccountAlreadyExists) {
		t.Errorf("el segundo Insert devolvió %v, se esperaba ErrAccountAlreadyExists", err)
	}
}

func TestAccountSqlLockByUserDevuelveLaCuentaDentroDeLaTransaccion(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)
	userID := seedUser(t, db)

	account := model.Account{
		UserID:           userID,
		AvailableBalance: model.CreditsFromUnits(990),
		ReservedBalance:  model.CreditsFromUnits(10),
	}
	if err := accountDao.Insert(t.Context(), nil, account); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	tx := beginTx(t, db)
	locked, err := accountDao.LockByUser(t.Context(), tx, userID)
	if err != nil {
		t.Fatalf("LockByUser devolvió %v", err)
	}
	if locked != account {
		t.Errorf("la cuenta bloqueada es %+v, se esperaba %+v", locked, account)
	}
}

func TestAccountSqlLockByUserSinTransaccionEsRechazado(t *testing.T) {
	accountDao := dao.NewAccountDao(nil)

	if _, err := accountDao.LockByUser(t.Context(), nil, 1); err == nil {
		t.Error("LockByUser sin transacción devolvió nil")
	}
}

func TestAccountSqlUpdateBalancesPersisteLosDosSaldos(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)
	userID := seedUser(t, db)

	if err := accountDao.Insert(t.Context(), nil, model.Account{
		UserID:           userID,
		AvailableBalance: model.CreditsFromUnits(1000),
	}); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	updated := model.Account{
		UserID:           userID,
		AvailableBalance: model.CreditsFromUnits(990),
		ReservedBalance:  model.CreditsFromUnits(10),
	}
	if err := accountDao.UpdateBalances(t.Context(), nil, updated); err != nil {
		t.Fatalf("UpdateBalances devolvió %v", err)
	}

	stored, err := accountDao.GetByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("GetByUser devolvió %v", err)
	}
	if stored != updated {
		t.Errorf("la cuenta leída es %+v, se esperaba %+v", stored, updated)
	}
}

func TestAccountSqlUpdateBalancesDeUnaCuentaInexistente(t *testing.T) {
	db := startPostgres(t)
	accountDao := dao.NewAccountDao(db)

	err := accountDao.UpdateBalances(t.Context(), nil, model.Account{UserID: seedUser(t, db)})
	if !errors.Is(err, model.ErrAccountNotFound) {
		t.Errorf("UpdateBalances devolvió %v, se esperaba ErrAccountNotFound", err)
	}
}

func TestAccountSqlConDbNilFalla(t *testing.T) {
	accountDao := dao.NewAccountDao(nil)

	if _, err := accountDao.GetByUser(t.Context(), 1); err == nil {
		t.Error("GetByUser con db nil devolvió nil")
	}
	if err := accountDao.Insert(t.Context(), nil, model.Account{UserID: 1}); err == nil {
		t.Error("Insert con db nil devolvió nil")
	}
}
