package dao_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestPositionSqlInsertIfAbsentCreaLaPosicion(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)
	position := model.Position{
		UserID:           seedUser(t, db),
		PlayerID:         seedPlayer(t, db),
		ReservedQuantity: 100,
	}

	inserted, err := positionDao.InsertIfAbsent(t.Context(), nil, position)
	if err != nil {
		t.Fatalf("InsertIfAbsent devolvió %v", err)
	}
	if !inserted {
		t.Fatal("InsertIfAbsent informó que no insertó una posición nueva")
	}

	positions, err := positionDao.ListByUser(t.Context(), position.UserID)
	if err != nil {
		t.Fatalf("ListByUser devolvió %v", err)
	}
	if len(positions) != 1 || positions[0] != position {
		t.Errorf("se leyó %+v, se esperaba una sola posición %+v", positions, position)
	}
}

func TestPositionSqlInsertIfAbsentNoEmiteUnSegundoLoteDeTokens(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)
	position := model.Position{
		UserID:           seedUser(t, db),
		PlayerID:         seedPlayer(t, db),
		ReservedQuantity: 100,
	}

	if _, err := positionDao.InsertIfAbsent(t.Context(), nil, position); err != nil {
		t.Fatalf("el primer InsertIfAbsent devolvió %v", err)
	}

	inserted, err := positionDao.InsertIfAbsent(t.Context(), nil, position)
	if err != nil {
		t.Fatalf("el segundo InsertIfAbsent devolvió %v", err)
	}
	if inserted {
		t.Error("el segundo InsertIfAbsent insertó una posición y duplicó la emisión")
	}

	total, err := positionDao.TotalQuantityByPlayer(t.Context(), position.PlayerID)
	if err != nil {
		t.Fatalf("TotalQuantityByPlayer devolvió %v", err)
	}
	if total != 100 {
		t.Errorf("hay %d tokens del jugador, se esperaban exactamente 100", total)
	}
}

func TestPositionSqlLockByUserAndPlayerDevuelveLaPosicion(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)
	position := model.Position{
		UserID:            seedUser(t, db),
		PlayerID:          seedPlayer(t, db),
		AvailableQuantity: 6,
		ReservedQuantity:  4,
		CostBasis:         model.CreditsFromUnits(10),
	}
	if _, err := positionDao.InsertIfAbsent(t.Context(), nil, position); err != nil {
		t.Fatalf("InsertIfAbsent devolvió %v", err)
	}

	tx := beginTx(t, db)
	locked, err := positionDao.LockByUserAndPlayer(t.Context(), tx, position.UserID, position.PlayerID)
	if err != nil {
		t.Fatalf("LockByUserAndPlayer devolvió %v", err)
	}
	if locked != position {
		t.Errorf("la posición bloqueada es %+v, se esperaba %+v", locked, position)
	}
}

func TestPositionSqlLockByUserAndPlayerSinPosicion(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)

	tx := beginTx(t, db)
	_, err := positionDao.LockByUserAndPlayer(t.Context(), tx, seedUser(t, db), seedPlayer(t, db))
	if !errors.Is(err, model.ErrPositionNotFound) {
		t.Errorf("LockByUserAndPlayer devolvió %v, se esperaba ErrPositionNotFound", err)
	}
}

func TestPositionSqlUpdatePersisteCantidadesYCosto(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)
	position := model.Position{
		UserID:            seedUser(t, db),
		PlayerID:          seedPlayer(t, db),
		AvailableQuantity: 10,
		CostBasis:         model.CreditsFromUnits(10),
	}
	if _, err := positionDao.InsertIfAbsent(t.Context(), nil, position); err != nil {
		t.Fatalf("InsertIfAbsent devolvió %v", err)
	}

	position.AvailableQuantity = 6
	position.ReservedQuantity = 0
	position.CostBasis = model.CreditsFromUnits(6)
	if err := positionDao.Update(t.Context(), nil, position); err != nil {
		t.Fatalf("Update devolvió %v", err)
	}

	positions, err := positionDao.ListByUser(t.Context(), position.UserID)
	if err != nil {
		t.Fatalf("ListByUser devolvió %v", err)
	}
	if positions[0] != position {
		t.Errorf("la posición leída es %+v, se esperaba %+v", positions[0], position)
	}
}

func TestPositionSqlUpdateDeUnaPosicionInexistente(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)

	err := positionDao.Update(t.Context(), nil, model.Position{
		UserID:   seedUser(t, db),
		PlayerID: seedPlayer(t, db),
	})
	if !errors.Is(err, model.ErrPositionNotFound) {
		t.Errorf("Update devolvió %v, se esperaba ErrPositionNotFound", err)
	}
}

func TestPositionSqlListByUserNoDevuelveLasDeOtro(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)
	playerID := seedPlayer(t, db)
	propia := model.Position{UserID: seedUser(t, db), PlayerID: playerID, AvailableQuantity: 5}
	ajena := model.Position{UserID: seedUser(t, db), PlayerID: playerID, AvailableQuantity: 7}

	for _, position := range []model.Position{propia, ajena} {
		if _, err := positionDao.InsertIfAbsent(t.Context(), nil, position); err != nil {
			t.Fatalf("InsertIfAbsent devolvió %v", err)
		}
	}

	positions, err := positionDao.ListByUser(t.Context(), propia.UserID)
	if err != nil {
		t.Fatalf("ListByUser devolvió %v", err)
	}
	if len(positions) != 1 || positions[0].UserID != propia.UserID {
		t.Errorf("ListByUser devolvió %+v, se esperaba sólo la posición propia", positions)
	}
}

func TestPositionSqlListByUserSinPosicionesDevuelveVacio(t *testing.T) {
	db := startPostgres(t)
	positionDao := dao.NewPositionDao(db)

	positions, err := positionDao.ListByUser(t.Context(), seedUser(t, db))
	if err != nil {
		t.Fatalf("ListByUser devolvió %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("ListByUser devolvió %+v, se esperaba vacío", positions)
	}
}
