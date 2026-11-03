package dao_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type mercadoSembrado struct {
	buyerID     int64
	sellerID    int64
	playerID    int64
	buyOrderID  int64
	sellOrderID int64
}

func sembrarMercado(t *testing.T, db *dao.OrderSql, buyerID, sellerID, playerID int64) mercadoSembrado {
	t.Helper()

	buyOrder := insertarOrden(t, db, nil, nuevaOrden(buyerID, playerID, model.SideBuy, model.CreditsFromUnits(1), 10))
	sellOrder := insertarOrden(t, db, nil, nuevaOrden(sellerID, playerID, model.SideSell, model.CreditsFromUnits(1), 10))

	return mercadoSembrado{
		buyerID:     buyerID,
		sellerID:    sellerID,
		playerID:    playerID,
		buyOrderID:  buyOrder.ID,
		sellOrderID: sellOrder.ID,
	}
}

func (m mercadoSembrado) operacion(quantity int64, price model.Credits) model.Trade {
	return model.Trade{
		BuyOrderID:  m.buyOrderID,
		SellOrderID: m.sellOrderID,
		BuyerID:     m.buyerID,
		SellerID:    m.sellerID,
		PlayerID:    m.playerID,
		Quantity:    quantity,
		Price:       price,
		TotalAmount: price.Times(quantity),
	}
}

func TestTradeSqlInsertDevuelveIdentificadorYFecha(t *testing.T) {
	db := startPostgres(t)
	tradeDao := dao.NewTradeDao(db)
	mercado := sembrarMercado(t, dao.NewOrderDao(db), seedUser(t, db), seedUser(t, db), seedPlayer(t, db))

	stored, err := tradeDao.Insert(t.Context(), nil, mercado.operacion(4, model.CreditsFromUnits(1)))
	if err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}
	if stored.ID == 0 || stored.ExecutedAt.IsZero() {
		t.Errorf("la operación insertada es %+v", stored)
	}
}

func TestTradeSqlListByParticipantDevuelveLoMasNuevoPrimero(t *testing.T) {
	db := startPostgres(t)
	tradeDao := dao.NewTradeDao(db)
	mercado := sembrarMercado(t, dao.NewOrderDao(db), seedUser(t, db), seedUser(t, db), seedPlayer(t, db))

	primera, err := tradeDao.Insert(t.Context(), nil, mercado.operacion(2, model.CreditsFromUnits(1)))
	if err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}
	segunda, err := tradeDao.Insert(t.Context(), nil, mercado.operacion(3, model.CreditsFromUnits(2)))
	if err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	trades, err := tradeDao.ListByParticipant(t.Context(), mercado.buyerID, 10, 0)
	if err != nil {
		t.Fatalf("ListByParticipant devolvió %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("vinieron %d operaciones, se esperaban 2", len(trades))
	}
	if trades[0].ID != segunda.ID || trades[1].ID != primera.ID {
		t.Errorf("el orden fue %d, %d y se esperaba %d, %d",
			trades[0].ID, trades[1].ID, segunda.ID, primera.ID)
	}
	if trades[0].TotalAmount != model.CreditsFromUnits(6) {
		t.Errorf("TotalAmount = %d, se esperaban 6 créditos", trades[0].TotalAmount)
	}
}

func TestTradeSqlListByParticipantAlcanzaALasDosPuntas(t *testing.T) {
	db := startPostgres(t)
	tradeDao := dao.NewTradeDao(db)
	mercado := sembrarMercado(t, dao.NewOrderDao(db), seedUser(t, db), seedUser(t, db), seedPlayer(t, db))

	if _, err := tradeDao.Insert(t.Context(), nil, mercado.operacion(2, model.CreditsFromUnits(1))); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	for _, userID := range []int64{mercado.buyerID, mercado.sellerID} {
		trades, err := tradeDao.ListByParticipant(t.Context(), userID, 10, 0)
		if err != nil {
			t.Fatalf("ListByParticipant devolvió %v", err)
		}
		if len(trades) != 1 {
			t.Errorf("el usuario %d vio %d operaciones, se esperaba 1", userID, len(trades))
		}
	}
}

func TestTradeSqlListByParticipantPagina(t *testing.T) {
	db := startPostgres(t)
	tradeDao := dao.NewTradeDao(db)
	mercado := sembrarMercado(t, dao.NewOrderDao(db), seedUser(t, db), seedUser(t, db), seedPlayer(t, db))

	for range 3 {
		if _, err := tradeDao.Insert(t.Context(), nil, mercado.operacion(1, model.CreditsFromUnits(1))); err != nil {
			t.Fatalf("Insert devolvió %v", err)
		}
	}

	total, err := tradeDao.CountByParticipant(t.Context(), mercado.buyerID)
	if err != nil {
		t.Fatalf("CountByParticipant devolvió %v", err)
	}
	if total != 3 {
		t.Errorf("CountByParticipant = %d, se esperaba 3", total)
	}

	page, err := tradeDao.ListByParticipant(t.Context(), mercado.buyerID, 2, 2)
	if err != nil {
		t.Fatalf("ListByParticipant devolvió %v", err)
	}
	if len(page) != 1 {
		t.Errorf("la segunda página trajo %d operaciones, se esperaba 1", len(page))
	}
}

func TestTradeSqlListByParticipantSinOperaciones(t *testing.T) {
	db := startPostgres(t)
	tradeDao := dao.NewTradeDao(db)

	trades, err := tradeDao.ListByParticipant(t.Context(), seedUser(t, db), 10, 0)
	if err != nil {
		t.Fatalf("ListByParticipant devolvió %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("vinieron %d operaciones, se esperaba vacío", len(trades))
	}
}
