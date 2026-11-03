package dao_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func nuevaOrden(userID, playerID int64, side model.OrderSide, price model.Credits, quantity int64) model.Order {
	return model.Order{
		Side:        side,
		TimeInForce: model.GoodTillCancelled,
		Status:      model.StatusPending,
		UserID:      userID,
		PlayerID:    playerID,
		Quantity:    quantity,
		Price:       price,
	}
}

func insertarOrden(t *testing.T, orderDao *dao.OrderSql, tx *sql.Tx, order model.Order) model.Order {
	t.Helper()

	stored, err := orderDao.Insert(t.Context(), tx, order)
	if err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	return stored
}

func TestOrderSqlInsertDevuelveElIdentificadorYLasFechas(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)

	stored := insertarOrden(t, orderDao, nil,
		nuevaOrden(seedUser(t, db), seedPlayer(t, db), model.SideBuy, model.CreditsFromUnits(1), 10))

	if stored.ID == 0 {
		t.Error("la orden insertada no tiene identificador")
	}
	if stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() {
		t.Error("la orden insertada no tiene fechas")
	}
}

func TestOrderSqlGetByIDDevuelveLaOrdenCompleta(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	order := nuevaOrden(seedUser(t, db), seedPlayer(t, db), model.SideSell, model.CreditsFromUnits(2), 7)
	order.MarketMaker = true

	stored := insertarOrden(t, orderDao, nil, order)

	read, err := orderDao.GetByID(t.Context(), nil, stored.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió %v", err)
	}
	if read.Side != model.SideSell || read.Price != model.CreditsFromUnits(2) ||
		read.Quantity != 7 || !read.MarketMaker || read.Status != model.StatusPending {
		t.Errorf("la orden leída es %+v", read)
	}
}

func TestOrderSqlGetByIDInexistente(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)

	if _, err := orderDao.GetByID(t.Context(), nil, 9999); !errors.Is(err, model.ErrOrderNotFound) {
		t.Errorf("GetByID devolvió %v, se esperaba ErrOrderNotFound", err)
	}
}

func TestOrderSqlUpdatePersisteEjecutadoYEstado(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	stored := insertarOrden(t, orderDao, nil,
		nuevaOrden(seedUser(t, db), seedPlayer(t, db), model.SideBuy, model.CreditsFromUnits(1), 10))

	stored.ExecutedQuantity = 4
	stored.Status = model.StatusPartiallyFilled
	if err := orderDao.Update(t.Context(), nil, stored); err != nil {
		t.Fatalf("Update devolvió %v", err)
	}

	read, err := orderDao.GetByID(t.Context(), nil, stored.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió %v", err)
	}
	if read.ExecutedQuantity != 4 || read.Status != model.StatusPartiallyFilled {
		t.Errorf("la orden leída es %+v, se esperaba 4 ejecutadas y PARTIALLY_FILLED", read)
	}
}

func TestOrderSqlUpdateDeUnaOrdenInexistente(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)

	err := orderDao.Update(t.Context(), nil, model.Order{ID: 9999, Status: model.StatusFilled})
	if !errors.Is(err, model.ErrOrderNotFound) {
		t.Errorf("Update devolvió %v, se esperaba ErrOrderNotFound", err)
	}
}

func TestOrderSqlLockLiveBookOrdenaPorPrecioYAntiguedad(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	cara := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(3), 5))
	barataVieja := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(1), 5))
	barataNueva := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(1), 5))

	tx := beginTx(t, db)
	book, err := orderDao.LockLiveBook(t.Context(), tx, playerID, model.SideSell)
	if err != nil {
		t.Fatalf("LockLiveBook devolvió %v", err)
	}

	want := []int64{barataVieja.ID, barataNueva.ID, cara.ID}
	if len(book) != len(want) {
		t.Fatalf("el libro trajo %d órdenes, se esperaban %d", len(book), len(want))
	}
	for i := range want {
		if book[i].ID != want[i] {
			t.Errorf("en la posición %d vino la orden %d, se esperaba la %d", i, book[i].ID, want[i])
		}
	}
}

func TestOrderSqlLockLiveBookOmiteLasFinalesYElOtroLado(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	cancelada := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(1), 5))
	cancelada.Status = model.StatusCancelled
	if err := orderDao.Update(t.Context(), nil, cancelada); err != nil {
		t.Fatalf("Update devolvió %v", err)
	}
	insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideBuy, model.CreditsFromUnits(1), 5))
	viva := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(2), 5))

	tx := beginTx(t, db)
	book, err := orderDao.LockLiveBook(t.Context(), tx, playerID, model.SideSell)
	if err != nil {
		t.Fatalf("LockLiveBook devolvió %v", err)
	}
	if len(book) != 1 || book[0].ID != viva.ID {
		t.Errorf("el libro trajo %+v, se esperaba sólo la orden viva de venta", book)
	}
}

func TestOrderSqlLockLiveBookSinTransaccionEsRechazado(t *testing.T) {
	orderDao := dao.NewOrderDao(nil)

	if _, err := orderDao.LockLiveBook(t.Context(), nil, 1, model.SideSell); err == nil {
		t.Error("LockLiveBook sin transacción devolvió nil")
	}
}

func TestOrderSqlListLiveByUserFiltraPorUsuarioYJugador(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	propio := seedUser(t, db)
	ajeno := seedUser(t, db)
	primerJugador := seedPlayer(t, db)
	segundoJugador := seedPlayer(t, db)

	delPrimero := insertarOrden(t, orderDao, nil, nuevaOrden(propio, primerJugador, model.SideBuy, model.CreditsFromUnits(1), 5))
	insertarOrden(t, orderDao, nil, nuevaOrden(propio, segundoJugador, model.SideBuy, model.CreditsFromUnits(1), 5))
	insertarOrden(t, orderDao, nil, nuevaOrden(ajeno, primerJugador, model.SideBuy, model.CreditsFromUnits(1), 5))

	todas, err := orderDao.ListLiveByUser(t.Context(), propio, nil)
	if err != nil {
		t.Fatalf("ListLiveByUser devolvió %v", err)
	}
	if len(todas) != 2 {
		t.Errorf("sin filtro vinieron %d órdenes, se esperaban 2", len(todas))
	}

	filtradas, err := orderDao.ListLiveByUser(t.Context(), propio, &primerJugador)
	if err != nil {
		t.Fatalf("ListLiveByUser con filtro devolvió %v", err)
	}
	if len(filtradas) != 1 || filtradas[0].ID != delPrimero.ID {
		t.Errorf("con filtro vinieron %+v, se esperaba sólo la del primer jugador", filtradas)
	}
}

func TestOrderSqlListLiveByUserOmiteLasFinales(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	completa := insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideBuy, model.CreditsFromUnits(1), 5))
	completa.ExecutedQuantity = 5
	completa.Status = model.StatusFilled
	if err := orderDao.Update(t.Context(), nil, completa); err != nil {
		t.Fatalf("Update devolvió %v", err)
	}

	orders, err := orderDao.ListLiveByUser(t.Context(), userID, nil)
	if err != nil {
		t.Fatalf("ListLiveByUser devolvió %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("ListLiveByUser devolvió %+v, se esperaba vacío", orders)
	}
}

func TestOrderSqlGetLiveMarketMakerEncuentraLaOfertaDelCreadorDeMercado(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	insertarOrden(t, orderDao, nil, nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(2), 5))
	marketMaker := nuevaOrden(userID, playerID, model.SideSell, model.CreditsFromUnits(1), 100)
	marketMaker.MarketMaker = true
	stored := insertarOrden(t, orderDao, nil, marketMaker)

	found, err := orderDao.GetLiveMarketMaker(t.Context(), nil, playerID, model.SideSell)
	if err != nil {
		t.Fatalf("GetLiveMarketMaker devolvió %v", err)
	}
	if found.ID != stored.ID {
		t.Errorf("se encontró la orden %d, se esperaba la %d", found.ID, stored.ID)
	}
}

func TestOrderSqlGetLiveMarketMakerSinOfertaViva(t *testing.T) {
	db := startPostgres(t)
	orderDao := dao.NewOrderDao(db)

	_, err := orderDao.GetLiveMarketMaker(t.Context(), nil, seedPlayer(t, db), model.SideSell)
	if !errors.Is(err, model.ErrOrderNotFound) {
		t.Errorf("GetLiveMarketMaker devolvió %v, se esperaba ErrOrderNotFound", err)
	}
}
