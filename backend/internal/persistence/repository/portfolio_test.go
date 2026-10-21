package repository_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func newPortfolioRepository(db *sql.DB) *repository.PortfolioRepository {
	return repository.NewPortfolioRepository(
		dao.NewAccountDao(db),
		dao.NewPositionDao(db),
		dao.NewQuoteDao(db),
		dao.NewTradeDao(db),
	)
}

func seedMarketTrade(t *testing.T, db *sql.DB, buyer, seller, playerID int64, quantity int64) model.Trade {
	t.Helper()

	price := model.CreditsFromUnits(1)
	buyOrder := seedMarketOrder(t, db, aRestingOrder(buyer, playerID, model.SideBuy, price, quantity))
	sellOrder := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideSell, price, quantity))

	trade, err := dao.NewTradeDao(db).Insert(t.Context(), nil, model.Trade{
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		BuyerID:     buyer,
		SellerID:    seller,
		PlayerID:    playerID,
		Quantity:    quantity,
		Price:       price,
		TotalAmount: price.Times(quantity),
	})
	if err != nil {
		t.Fatalf("no se pudo sembrar la operación: %v", err)
	}

	return trade
}

func TestPortfolioRepository(t *testing.T) {
	db := startPostgres(t)
	repo := newPortfolioRepository(db)

	t.Run("lee la cuenta del usuario y avisa cuando no tiene", func(t *testing.T) {
		assertAccountIsReadBack(t, db, repo)
	})

	t.Run("lista las posiciones del usuario y solo las suyas", func(t *testing.T) {
		assertPositionsBelongToTheirOwner(t, db, repo)
	})

	t.Run("la cotizacion vigente es la ultima de cada jugador", func(t *testing.T) {
		assertLatestPriceIsTheMostRecentOne(t, db, repo)
	})

	t.Run("el historial pagina de la mas nueva a la mas vieja", func(t *testing.T) {
		assertTradeHistoryIsPaginatedNewestFirst(t, db, repo)
	})

	t.Run("el historial incluye las compras y las ventas del usuario", func(t *testing.T) {
		assertTradeHistoryCoversBothSides(t, db, repo)
	})

	t.Run("un usuario sin operaciones recibe una pagina vacia", func(t *testing.T) {
		assertEmptyTradeHistory(t, db, repo)
	})
}

func assertAccountIsReadBack(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	userID := seedMarketUser(t, db)
	account := model.Account{
		UserID:           userID,
		AvailableBalance: model.CreditsFromUnits(850),
		ReservedBalance:  model.CreditsFromUnits(150),
	}
	seedMarketAccount(t, db, account)

	stored, err := repo.Account(t.Context(), userID)
	if err != nil {
		t.Fatalf("Account devolvió %v", err)
	}
	if stored != account {
		t.Errorf("la cuenta leída es %+v, se esperaba %+v", stored, account)
	}

	if _, missingErr := repo.Account(t.Context(), seedMarketUser(t, db)); !errors.Is(missingErr, model.ErrAccountNotFound) {
		t.Errorf("Account devolvió %v, se esperaba ErrAccountNotFound", missingErr)
	}
}

func assertPositionsBelongToTheirOwner(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	owner := seedMarketUser(t, db)
	stranger := seedMarketUser(t, db)
	firstPlayer := seedMarketPlayer(t, db)
	secondPlayer := seedMarketPlayer(t, db)

	seedMarketPosition(t, db, model.Position{UserID: owner, PlayerID: firstPlayer, AvailableQuantity: 3, CostBasis: model.CreditsFromUnits(3)})
	seedMarketPosition(t, db, model.Position{UserID: owner, PlayerID: secondPlayer, ReservedQuantity: 2, CostBasis: model.CreditsFromUnits(4)})
	seedMarketPosition(t, db, model.Position{UserID: stranger, PlayerID: firstPlayer, AvailableQuantity: 9, CostBasis: model.CreditsFromUnits(9)})

	positions, err := repo.Positions(t.Context(), owner)
	if err != nil {
		t.Fatalf("Positions devolvió %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("Positions devolvió %d posiciones, se esperaban 2: %+v", len(positions), positions)
	}
	for _, position := range positions {
		if position.UserID != owner {
			t.Errorf("Positions devolvió una posición del usuario %d", position.UserID)
		}
	}

	empty, err := repo.Positions(t.Context(), seedMarketUser(t, db))
	if err != nil {
		t.Fatalf("Positions devolvió %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("un usuario sin posiciones recibió %+v", empty)
	}
}

func assertLatestPriceIsTheMostRecentOne(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	quotedPlayer := seedMarketPlayer(t, db)
	silentPlayer := seedMarketPlayer(t, db)

	seedMarketQuote(t, db, quotedPlayer, model.CreditsFromUnits(1))
	if _, err := dao.NewQuoteDao(db).Insert(t.Context(), nil, model.PlayerQuote{
		PlayerID:        quotedPlayer,
		Price:           model.CreditsFromUnits(3),
		Strategy:        "performance",
		StrategyVersion: "1",
	}); err != nil {
		t.Fatalf("no se pudo sembrar la segunda cotización: %v", err)
	}

	prices, err := repo.LatestPrices(t.Context(), []int64{quotedPlayer, silentPlayer})
	if err != nil {
		t.Fatalf("LatestPrices devolvió %v", err)
	}
	if prices[quotedPlayer] != model.CreditsFromUnits(3) {
		t.Errorf("la cotización vigente es %d, se esperaban 3 créditos", prices[quotedPlayer])
	}
	if _, quoted := prices[silentPlayer]; quoted {
		t.Errorf("un jugador sin cotizaciones apareció con %d", prices[silentPlayer])
	}
}

func assertTradeHistoryIsPaginatedNewestFirst(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	buyer := seedMarketUser(t, db)
	seller := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)

	first := seedMarketTrade(t, db, buyer, seller, playerID, 1)
	second := seedMarketTrade(t, db, buyer, seller, playerID, 2)
	third := seedMarketTrade(t, db, buyer, seller, playerID, 3)

	page, err := repo.Trades(t.Context(), buyer, 1, 2)
	if err != nil {
		t.Fatalf("Trades devolvió %v", err)
	}
	if page.Total != 3 || page.TotalPages != 2 || page.Page != 1 || page.Limit != 2 {
		t.Errorf("la página es %+v, se esperaban 3 operaciones en 2 páginas", page)
	}
	if len(page.Items) != 2 || page.Items[0].ID != third.ID || page.Items[1].ID != second.ID {
		t.Errorf("la primera página trajo %+v, se esperaban las operaciones %d y %d", page.Items, third.ID, second.ID)
	}

	last, err := repo.Trades(t.Context(), buyer, 2, 2)
	if err != nil {
		t.Fatalf("Trades devolvió %v", err)
	}
	if len(last.Items) != 1 || last.Items[0].ID != first.ID {
		t.Errorf("la segunda página trajo %+v, se esperaba sólo la operación %d", last.Items, first.ID)
	}
}

func assertTradeHistoryCoversBothSides(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	user := seedMarketUser(t, db)
	counterparty := seedMarketUser(t, db)
	stranger := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)

	bought := seedMarketTrade(t, db, user, counterparty, playerID, 1)
	sold := seedMarketTrade(t, db, counterparty, user, playerID, 2)
	seedMarketTrade(t, db, counterparty, stranger, playerID, 3)

	page, err := repo.Trades(t.Context(), user, 1, 10)
	if err != nil {
		t.Fatalf("Trades devolvió %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("el historial trajo %d operaciones, se esperaban 2: %+v", page.Total, page.Items)
	}

	seen := map[int64]bool{}
	for _, trade := range page.Items {
		seen[trade.ID] = true
	}
	if !seen[bought.ID] || !seen[sold.ID] {
		t.Errorf("el historial es %+v, se esperaban la compra %d y la venta %d", page.Items, bought.ID, sold.ID)
	}
}

func assertEmptyTradeHistory(t *testing.T, db *sql.DB, repo *repository.PortfolioRepository) {
	t.Helper()

	page, err := repo.Trades(t.Context(), seedMarketUser(t, db), 0, 0)
	if err != nil {
		t.Fatalf("Trades devolvió %v", err)
	}
	if page.Total != 0 || page.TotalPages != 0 || len(page.Items) != 0 {
		t.Errorf("la página vacía es %+v", page)
	}
	if page.Page != 1 || page.Limit != 20 {
		t.Errorf("la página vacía quedó en %d con límite %d, se esperaba la primera con 20", page.Page, page.Limit)
	}
}
