package dao_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestPlayerSqlLockPlayerTomaLaFila(t *testing.T) {
	db := startPostgres(t)
	playerDao := dao.NewPlayerDao(db)
	playerID := seedPlayer(t, db)

	tx := beginTx(t, db)
	active, err := playerDao.LockPlayer(t.Context(), tx, playerID)
	if err != nil {
		t.Fatalf("LockPlayer devolvió %v", err)
	}
	if !active {
		t.Error("LockPlayer informó inactivo a un jugador recién sembrado")
	}
}

func TestPlayerSqlLockPlayerInformaAlJugadorInactivo(t *testing.T) {
	db := startPostgres(t)
	playerDao := dao.NewPlayerDao(db)
	playerID := seedPlayer(t, db)

	if _, err := db.ExecContext(t.Context(), `UPDATE players SET active = FALSE WHERE id = $1`, playerID); err != nil {
		t.Fatalf("no se pudo desactivar el jugador: %v", err)
	}

	tx := beginTx(t, db)
	active, err := playerDao.LockPlayer(t.Context(), tx, playerID)
	if err != nil {
		t.Fatalf("LockPlayer devolvió %v", err)
	}
	if active {
		t.Error("LockPlayer informó activo a un jugador desactivado")
	}
}

func TestPlayerSqlLockPlayerSerializaALosQueCompiten(t *testing.T) {
	db := startPostgres(t)
	playerDao := dao.NewPlayerDao(db)
	playerID := seedPlayer(t, db)

	first, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("no se pudo abrir la primera transacción: %v", err)
	}
	defer func() { _ = first.Rollback() }()

	if _, err = playerDao.LockPlayer(t.Context(), first, playerID); err != nil {
		t.Fatalf("el primer LockPlayer devolvió %v", err)
	}

	blockedCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	second, err := db.BeginTx(blockedCtx, nil)
	if err != nil {
		t.Fatalf("no se pudo abrir la segunda transacción: %v", err)
	}
	defer func() { _ = second.Rollback() }()

	_, err = playerDao.LockPlayer(blockedCtx, second, playerID)
	if err == nil {
		t.Fatal("el segundo LockPlayer obtuvo el lock mientras la primera transacción lo tenía tomado")
	}
	if !errors.Is(blockedCtx.Err(), context.DeadlineExceeded) {
		t.Errorf("el segundo LockPlayer falló con %v antes de que venciera el plazo: se esperaba que quedara esperando el lock", err)
	}
}

func TestPlayerSqlLockPlayerDeUnJugadorInexistente(t *testing.T) {
	db := startPostgres(t)
	playerDao := dao.NewPlayerDao(db)

	tx := beginTx(t, db)
	if _, err := playerDao.LockPlayer(t.Context(), tx, 9999); !errors.Is(err, dao.ErrNotFound) {
		t.Errorf("LockPlayer devolvió %v, se esperaba ErrNotFound", err)
	}
}

func TestPlayerSqlLockPlayerSinTransaccionEsRechazado(t *testing.T) {
	playerDao := dao.NewPlayerDao(nil)

	if _, err := playerDao.LockPlayer(t.Context(), nil, 1); err == nil {
		t.Error("LockPlayer sin transacción devolvió nil")
	}
}

func TestPlayerSqlListActiveIDsSoloTraeLosActivos(t *testing.T) {
	db := startPostgres(t)
	playerDao := dao.NewPlayerDao(db)
	activo := seedPlayer(t, db)
	inactivo := seedPlayer(t, db)

	if _, err := db.ExecContext(t.Context(), `UPDATE players SET active = FALSE WHERE id = $1`, inactivo); err != nil {
		t.Fatalf("no se pudo desactivar el jugador: %v", err)
	}

	ids, err := playerDao.ListActiveIDs(t.Context())
	if err != nil {
		t.Fatalf("ListActiveIDs devolvió %v", err)
	}
	if len(ids) != 1 || ids[0] != activo {
		t.Errorf("ListActiveIDs devolvió %v, se esperaba sólo %d", ids, activo)
	}
}
