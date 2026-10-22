package service_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func buyCommand(userID, playerID, quantity int64) service.PlaceOrderCommand {
	return service.PlaceOrderCommand{
		Actor:    "comprador@test.local",
		Side:     model.SideBuy,
		UserID:   userID,
		PlayerID: playerID,
		Quantity: quantity,
	}
}

func withPrice(command service.PlaceOrderCommand, price model.Credits) service.PlaceOrderCommand {
	command.Price = &price

	return command
}

func TestPlaceOrderSinPrecioUsaLaCotizacionVigenteYEjecutaCompleto(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 100)

	placed, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10))
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusFilled {
		t.Errorf("la orden quedó en %s, se esperaba FILLED", placed.Order.Status)
	}
	if placed.Order.TimeInForce != model.FillOrKill {
		t.Errorf("la vigencia por defecto fue %s, se esperaba FOK sin precio", placed.Order.TimeInForce)
	}
	if len(placed.Trades) != 1 {
		t.Fatalf("se registraron %d operaciones, se esperaba 1", len(placed.Trades))
	}

	buyerAccount := fixture.account(t, buyer)
	if buyerAccount.AvailableBalance != model.CreditsFromUnits(990) {
		t.Errorf("el comprador quedó con %d, se esperaban 990 créditos", buyerAccount.AvailableBalance)
	}
	if buyerAccount.ReservedBalance != 0 {
		t.Errorf("el comprador quedó con %d reservados, se esperaba cero", buyerAccount.ReservedBalance)
	}

	sellerAccount := fixture.account(t, seller)
	if sellerAccount.AvailableBalance != model.CreditsFromUnits(10) {
		t.Errorf("el vendedor quedó con %d, se esperaban 10 créditos", sellerAccount.AvailableBalance)
	}

	if quantity := fixture.position(t, buyer, playerID).AvailableQuantity; quantity != 10 {
		t.Errorf("el comprador quedó con %d tokens, se esperaban 10", quantity)
	}
	if remaining := fixture.position(t, seller, playerID).ReservedQuantity; remaining != 90 {
		t.Errorf("al vendedor le quedaron %d tokens reservados, se esperaban 90", remaining)
	}

	if !fixture.ledgerIsBalanced(t) {
		t.Error("hay una liquidación del libro mayor que no suma cero")
	}
}

func TestPlaceOrderSinCreditosSuficientesNoDejaRastro(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(5))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 100)

	_, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10))
	if !errors.Is(err, model.ErrInsufficientFunds) {
		t.Fatalf("PlaceOrder devolvió %v, se esperaba ErrInsufficientFunds", err)
	}

	buyerAccount := fixture.account(t, buyer)
	if buyerAccount.AvailableBalance != model.CreditsFromUnits(5) {
		t.Errorf("el comprador quedó con %d, se esperaban los 5 créditos intactos", buyerAccount.AvailableBalance)
	}
	if buyerAccount.ReservedBalance != 0 {
		t.Errorf("quedaron %d créditos reservados por una orden que nunca existió", buyerAccount.ReservedBalance)
	}
	if quantity := fixture.position(t, buyer, playerID).AvailableQuantity; quantity != 0 {
		t.Errorf("el comprador quedó con %d tokens tras una orden rechazada", quantity)
	}
}

func TestPlaceOrderEjecutaAlPrecioDeLaOrdenEnReposoYDevuelveLaDiferencia(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(2), 10)

	command := withPrice(buyCommand(buyer, playerID, 10), model.CreditsFromUnits(5))

	placed, err := fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if len(placed.Trades) != 1 {
		t.Fatalf("se registraron %d operaciones, se esperaba 1", len(placed.Trades))
	}
	if placed.Trades[0].Price != model.CreditsFromUnits(2) {
		t.Errorf("la ejecución se valuó a %d, se esperaba el precio en reposo de 2", placed.Trades[0].Price)
	}

	buyerAccount := fixture.account(t, buyer)
	if buyerAccount.AvailableBalance != model.CreditsFromUnits(980) {
		t.Errorf("el comprador quedó con %d, se esperaban 980: reservó 50 y gastó 20", buyerAccount.AvailableBalance)
	}
	if buyerAccount.ReservedBalance != 0 {
		t.Errorf("quedaron %d créditos reservados tras una orden completa", buyerAccount.ReservedBalance)
	}
}

func TestPlaceOrderConsumeElMasBaratoPrimeroYRegistraUnaOperacionPorOrden(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	caro := fixture.seedUser(t, 0)
	barato := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, caro, playerID, model.CreditsFromUnits(4), 5)
	fixture.seedRestingSell(t, barato, playerID, model.CreditsFromUnits(2), 5)

	command := withPrice(buyCommand(buyer, playerID, 10), model.CreditsFromUnits(5))

	placed, err := fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusFilled {
		t.Errorf("la orden quedó en %s, se esperaba FILLED", placed.Order.Status)
	}
	if len(placed.Trades) != 2 {
		t.Fatalf("se registraron %d operaciones, se esperaban 2", len(placed.Trades))
	}
	if placed.Trades[0].Price != model.CreditsFromUnits(2) {
		t.Errorf("la primera ejecución fue a %d, se esperaba la más barata primero", placed.Trades[0].Price)
	}
	if placed.Trades[1].Price != model.CreditsFromUnits(4) {
		t.Errorf("la segunda ejecución fue a %d, se esperaba la más cara después", placed.Trades[1].Price)
	}

	if !fixture.ledgerIsBalanced(t) {
		t.Error("hay una liquidación del libro mayor que no suma cero")
	}
}

func TestPlaceOrderRechazaLaCantidadYElPrecioNoPositivos(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	casos := map[string]struct {
		esperado error
		command  service.PlaceOrderCommand
	}{
		"cantidad cero":     {service.ErrInvalidQuantity, buyCommand(buyer, playerID, 0)},
		"cantidad negativa": {service.ErrInvalidQuantity, buyCommand(buyer, playerID, -3)},
		"precio cero":       {service.ErrInvalidPrice, withPrice(buyCommand(buyer, playerID, 1), 0)},
		"precio negativo":   {service.ErrInvalidPrice, withPrice(buyCommand(buyer, playerID, 1), -10)},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := fixture.market.PlaceOrder(t.Context(), caso.command); !errors.Is(err, caso.esperado) {
				t.Errorf("PlaceOrder devolvió %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}

func TestPlaceOrderRechazaAlJugadorDesconocidoYAlInactivo(t *testing.T) {
	fixture := newMarketFixture(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, 999999, 1)); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("con un jugador inexistente PlaceOrder devolvió %v, se esperaba ErrNotFound", err)
	}

	inactivo := fixture.seedPlayer(t)
	fixture.seedQuote(t, inactivo, model.CreditsFromUnits(1))
	fixture.deactivatePlayer(t, inactivo)

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, inactivo, 1)); !errors.Is(err, service.ErrPlayerInactive) {
		t.Errorf("con un jugador inactivo PlaceOrder devolvió %v, se esperaba ErrPlayerInactive", err)
	}

	account := fixture.account(t, buyer)
	if account.AvailableBalance != model.CreditsFromUnits(1000) || account.ReservedBalance != 0 {
		t.Errorf("la cuenta quedó en %d disponibles y %d reservados tras dos rechazos", account.AvailableBalance, account.ReservedBalance)
	}
}

func sellCommand(userID, playerID, quantity int64) service.PlaceOrderCommand {
	return service.PlaceOrderCommand{
		Actor:    "vendedor@test.local",
		Side:     model.SideSell,
		UserID:   userID,
		PlayerID: playerID,
		Quantity: quantity,
	}
}

func TestPlaceOrderDeVentaEjecutaContraLaCompraEnReposoYCobraElPrecio(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))
	fixture.seedRestingBuy(t, buyer, playerID, model.CreditsFromUnits(1), 100)

	placed, err := fixture.market.PlaceOrder(t.Context(), sellCommand(seller, playerID, 4))
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusFilled {
		t.Errorf("la orden quedó en %s, se esperaba FILLED", placed.Order.Status)
	}

	if balance := fixture.account(t, seller).AvailableBalance; balance != model.CreditsFromUnits(4) {
		t.Errorf("el vendedor quedó con %d, se esperaban 4 créditos", balance)
	}

	sellerPosition := fixture.position(t, seller, playerID)
	if sellerPosition.TotalQuantity() != 6 {
		t.Errorf("al vendedor le quedaron %d tokens, se esperaban 6", sellerPosition.TotalQuantity())
	}

	if quantity := fixture.position(t, buyer, playerID).AvailableQuantity; quantity != 4 {
		t.Errorf("el comprador recibió %d tokens, se esperaban 4", quantity)
	}

	if !fixture.ledgerIsBalanced(t) {
		t.Error("hay una liquidación del libro mayor que no suma cero")
	}
}

func TestPlaceOrderDeVentaSinTokensSuficientesEsRechazada(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 3, model.CreditsFromUnits(3))
	fixture.seedRestingBuy(t, buyer, playerID, model.CreditsFromUnits(1), 100)

	_, err := fixture.market.PlaceOrder(t.Context(), sellCommand(seller, playerID, 10))
	if !errors.Is(err, model.ErrInsufficientTokens) {
		t.Fatalf("PlaceOrder devolvió %v, se esperaba ErrInsufficientTokens", err)
	}

	position := fixture.position(t, seller, playerID)
	if position.AvailableQuantity != 3 || position.ReservedQuantity != 0 {
		t.Errorf("la posición quedó en %d disponibles y %d reservados, se esperaban 3 y 0",
			position.AvailableQuantity, position.ReservedQuantity)
	}
}

func TestPlaceOrderDeVentaNoPuedeUsarLosTokensYaReservados(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))

	primera := withPrice(sellCommand(seller, playerID, 6), model.CreditsFromUnits(50))
	if _, err := fixture.market.PlaceOrder(t.Context(), primera); err != nil {
		t.Fatalf("la primera venta devolvió %v", err)
	}

	segunda := withPrice(sellCommand(seller, playerID, 6), model.CreditsFromUnits(50))
	if _, err := fixture.market.PlaceOrder(t.Context(), segunda); !errors.Is(err, model.ErrInsufficientTokens) {
		t.Fatalf("la segunda venta devolvió %v, se esperaba ErrInsufficientTokens", err)
	}

	position := fixture.position(t, seller, playerID)
	if position.AvailableQuantity != 4 || position.ReservedQuantity != 6 {
		t.Errorf("la posición quedó en %d disponibles y %d reservados, se esperaban 4 y 6",
			position.AvailableQuantity, position.ReservedQuantity)
	}
}

func TestPlaceOrderDeVentaDejaIntactoElPrecioPromedioDeCompra(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(30))
	fixture.seedRestingBuy(t, buyer, playerID, model.CreditsFromUnits(9), 100)

	posicionPrevia := fixture.position(t, seller, playerID)
	antes := posicionPrevia.AveragePrice()

	if _, err := fixture.market.PlaceOrder(t.Context(), sellCommand(seller, playerID, 4)); err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	position := fixture.position(t, seller, playerID)
	if despues := position.AveragePrice(); despues != antes {
		t.Errorf("el precio promedio pasó de %d a %d: la venta no debe moverlo", antes, despues)
	}
	if position.CostBasis != model.CreditsFromUnits(18) {
		t.Errorf("el costo quedó en %d, se esperaban 18: seis tokens a un promedio de 3", position.CostBasis)
	}
}

func TestPlaceOrderDeVentaPorEncimaDelLibroQuedaEnReposoConLosTokensReservados(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))
	fixture.seedRestingBuy(t, buyer, playerID, model.CreditsFromUnits(2), 5)

	command := withPrice(sellCommand(seller, playerID, 10), model.CreditsFromUnits(50))

	placed, err := fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusPending {
		t.Errorf("la orden quedó en %s, se esperaba PENDING", placed.Order.Status)
	}
	if len(placed.Trades) != 0 {
		t.Errorf("se registraron %d operaciones, se esperaba ninguna", len(placed.Trades))
	}
	if fixture.orderStatus(t, placed.Order.ID) != model.StatusPending {
		t.Error("la orden no quedó en reposo en la base")
	}

	position := fixture.position(t, seller, playerID)
	if position.AvailableQuantity != 0 || position.ReservedQuantity != 10 {
		t.Errorf("la posición quedó en %d disponibles y %d reservados, se esperaban 0 y 10",
			position.AvailableQuantity, position.ReservedQuantity)
	}
}

func withTimeInForce(command service.PlaceOrderCommand, timeInForce model.TimeInForce) service.PlaceOrderCommand {
	command.TimeInForce = &timeInForce

	return command
}

type libroEscaso struct {
	fixture  *marketFixture
	playerID int64
	buyer    int64
	seller   int64
}

func nuevoLibroEscaso(t *testing.T) libroEscaso {
	t.Helper()

	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 5)

	return libroEscaso{fixture: fixture, playerID: playerID, buyer: buyer, seller: seller}
}

func TestPlaceOrderFillOrKillNoDejaNiOrdenNiReserva(t *testing.T) {
	libro := nuevoLibroEscaso(t)

	_, err := libro.fixture.market.PlaceOrder(t.Context(), buyCommand(libro.buyer, libro.playerID, 10))
	if !errors.Is(err, service.ErrNoLiquidity) {
		t.Fatalf("PlaceOrder devolvió %v, se esperaba ErrNoLiquidity", err)
	}

	account := libro.fixture.account(t, libro.buyer)
	if account.AvailableBalance != model.CreditsFromUnits(1000) || account.ReservedBalance != 0 {
		t.Errorf("la cuenta quedó en %d disponibles y %d reservados, se esperaban 1000 y 0",
			account.AvailableBalance, account.ReservedBalance)
	}
	if orders := libro.fixture.countOrders(t, libro.buyer); orders != 0 {
		t.Errorf("quedaron %d órdenes de una compra que se deshizo entera", orders)
	}
	if trades := libro.fixture.countTrades(t, libro.playerID); trades != 0 {
		t.Errorf("quedaron %d operaciones de una compra que se deshizo entera", trades)
	}
	posicion := libro.fixture.position(t, libro.buyer, libro.playerID)
	if quantity := posicion.TotalQuantity(); quantity != 0 {
		t.Errorf("el comprador quedó con %d tokens tras un FOK rechazado", quantity)
	}
}

func TestPlaceOrderGoodTillCancelledDejaElRemanenteEnReposoConSuReserva(t *testing.T) {
	libro := nuevoLibroEscaso(t)

	command := withPrice(buyCommand(libro.buyer, libro.playerID, 10), model.CreditsFromUnits(2))

	placed, err := libro.fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusPartiallyFilled {
		t.Errorf("la orden quedó en %s, se esperaba PARTIALLY_FILLED", placed.Order.Status)
	}
	if placed.Order.ExecutedQuantity != 5 {
		t.Errorf("se ejecutaron %d tokens, se esperaban 5", placed.Order.ExecutedQuantity)
	}
	if libro.fixture.orderStatus(t, placed.Order.ID) != model.StatusPartiallyFilled {
		t.Error("la orden no quedó PARTIALLY_FILLED en la base")
	}

	account := libro.fixture.account(t, libro.buyer)
	if account.ReservedBalance != model.CreditsFromUnits(10) {
		t.Errorf("quedaron %d créditos reservados, se esperaban 10: cinco tokens al límite de 2", account.ReservedBalance)
	}
	if account.AvailableBalance != model.CreditsFromUnits(985) {
		t.Errorf("quedaron %d disponibles, se esperaban 985: reservó 20, gastó 5 y retiene 10", account.AvailableBalance)
	}
}

func TestPlaceOrderImmediateOrCancelLiberaLaReservaDelRemanente(t *testing.T) {
	libro := nuevoLibroEscaso(t)

	command := withTimeInForce(
		withPrice(buyCommand(libro.buyer, libro.playerID, 10), model.CreditsFromUnits(2)),
		model.ImmediateOrCancel,
	)

	placed, err := libro.fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if placed.Order.Status != model.StatusCancelled {
		t.Errorf("la orden quedó en %s, se esperaba CANCELLED", placed.Order.Status)
	}
	if placed.Order.ExecutedQuantity != 5 {
		t.Errorf("se ejecutaron %d tokens, se esperaban 5", placed.Order.ExecutedQuantity)
	}
	if libro.fixture.orderStatus(t, placed.Order.ID) != model.StatusCancelled {
		t.Error("la orden no quedó CANCELLED en la base")
	}

	account := libro.fixture.account(t, libro.buyer)
	if account.ReservedBalance != 0 {
		t.Errorf("quedaron %d créditos reservados por un remanente que se cancelo", account.ReservedBalance)
	}
	if account.AvailableBalance != model.CreditsFromUnits(995) {
		t.Errorf("quedaron %d disponibles, se esperaban 995: solo se gastaron los 5 ejecutados", account.AvailableBalance)
	}
}

func TestPlaceOrderSinLibroNoEjecutaNada(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10)); !errors.Is(err, service.ErrNoLiquidity) {
		t.Fatalf("con el libro vacío PlaceOrder devolvió %v, se esperaba ErrNoLiquidity", err)
	}

	command := withPrice(buyCommand(buyer, playerID, 10), model.CreditsFromUnits(3))

	placed, err := fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("con el libro vacío y precio explícito PlaceOrder devolvió %v", err)
	}
	if placed.Order.Status != model.StatusPending {
		t.Errorf("la orden quedó en %s, se esperaba PENDING", placed.Order.Status)
	}
	if len(placed.Trades) != 0 {
		t.Errorf("se registraron %d operaciones contra un libro vacío", len(placed.Trades))
	}
	if reserved := fixture.account(t, buyer).ReservedBalance; reserved != model.CreditsFromUnits(30) {
		t.Errorf("quedaron %d créditos reservados, se esperaban 30 por la orden en reposo", reserved)
	}
}

func cancelCommand(userID, orderID int64) service.CancelOrderCommand {
	return service.CancelOrderCommand{
		Actor:   "cancelador@test.local",
		UserID:  userID,
		OrderID: orderID,
	}
}

func TestCancelOrderDeUnaCompraParcialLiberaSoloElRemanente(t *testing.T) {
	libro := nuevoLibroEscaso(t)

	command := withPrice(buyCommand(libro.buyer, libro.playerID, 10), model.CreditsFromUnits(2))

	placed, err := libro.fixture.market.PlaceOrder(t.Context(), command)
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}
	if placed.Order.Status != model.StatusPartiallyFilled {
		t.Fatalf("la orden quedó en %s, se esperaba PARTIALLY_FILLED para poder cancelarla", placed.Order.Status)
	}

	cancelled, err := libro.fixture.market.CancelOrder(t.Context(), cancelCommand(libro.buyer, placed.Order.ID))
	if err != nil {
		t.Fatalf("CancelOrder devolvió %v", err)
	}

	if cancelled.Order.Status != model.StatusCancelled {
		t.Errorf("la orden quedó en %s, se esperaba CANCELLED", cancelled.Order.Status)
	}
	if cancelled.Order.ExecutedQuantity != 5 {
		t.Errorf("la orden cancelada informa %d ejecutados, se esperaban 5", cancelled.Order.ExecutedQuantity)
	}
	if cancelled.ReleasedCredits != model.CreditsFromUnits(10) {
		t.Errorf("se liberaron %d créditos, se esperaban 10 por los cinco tokens pendientes", cancelled.ReleasedCredits)
	}

	account := libro.fixture.account(t, libro.buyer)
	if account.ReservedBalance != 0 {
		t.Errorf("quedaron %d créditos reservados por una orden cancelada", account.ReservedBalance)
	}
	if account.AvailableBalance != model.CreditsFromUnits(995) {
		t.Errorf("quedaron %d disponibles, se esperaban 995: solo se gastaron los 5 ejecutados", account.AvailableBalance)
	}
	if libro.fixture.orderStatus(t, placed.Order.ID) != model.StatusCancelled {
		t.Error("la orden no quedó CANCELLED en la base")
	}
}

func TestCancelOrderDeUnaVentaDevuelveLosTokensReservados(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))

	placed, err := fixture.market.PlaceOrder(t.Context(), withPrice(sellCommand(seller, playerID, 7), model.CreditsFromUnits(50)))
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	cancelled, err := fixture.market.CancelOrder(t.Context(), cancelCommand(seller, placed.Order.ID))
	if err != nil {
		t.Fatalf("CancelOrder devolvió %v", err)
	}

	if cancelled.ReleasedTokens != 7 {
		t.Errorf("se liberaron %d tokens, se esperaban 7", cancelled.ReleasedTokens)
	}

	position := fixture.position(t, seller, playerID)
	if position.AvailableQuantity != 10 || position.ReservedQuantity != 0 {
		t.Errorf("la posición quedó en %d disponibles y %d reservados, se esperaban 10 y 0",
			position.AvailableQuantity, position.ReservedQuantity)
	}
}

func TestCancelOrderRechazaAlQueNoEsDuenoYALaOrdenYaFinal(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	owner := fixture.seedUser(t, model.CreditsFromUnits(1000))
	intruso := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))

	placed, err := fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(owner, playerID, 3), model.CreditsFromUnits(2)))
	if err != nil {
		t.Fatalf("PlaceOrder devolvió %v", err)
	}

	if _, err := fixture.market.CancelOrder(t.Context(), cancelCommand(intruso, placed.Order.ID)); !errors.Is(err, service.ErrNotOwner) {
		t.Errorf("cancelar la orden de otro devolvió %v, se esperaba ErrNotOwner", err)
	}

	if _, err := fixture.market.CancelOrder(t.Context(), cancelCommand(owner, placed.Order.ID)); err != nil {
		t.Fatalf("la primera cancelación devolvió %v", err)
	}

	if _, err := fixture.market.CancelOrder(t.Context(), cancelCommand(owner, placed.Order.ID)); !errors.Is(err, model.ErrOrderNotLive) {
		t.Errorf("cancelar dos veces devolvió %v, se esperaba ErrOrderNotLive", err)
	}

	if _, err := fixture.market.CancelOrder(t.Context(), cancelCommand(owner, 999999)); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("cancelar una orden inexistente devolvió %v, se esperaba ErrNotFound", err)
	}
}

func TestListOpenOrdersNuncaDevuelveLasDeOtroUsuario(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	otroPlayer := fixture.seedPlayer(t)
	propio := fixture.seedUser(t, model.CreditsFromUnits(1000))
	ajeno := fixture.seedUser(t, model.CreditsFromUnits(1000))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedQuote(t, otroPlayer, model.CreditsFromUnits(1))

	propia, err := fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(propio, playerID, 3), model.CreditsFromUnits(2)))
	if err != nil {
		t.Fatalf("la orden propia devolvió %v", err)
	}
	if _, err = fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(propio, otroPlayer, 2), model.CreditsFromUnits(2))); err != nil {
		t.Fatalf("la segunda orden propia devolvió %v", err)
	}
	if _, err = fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(ajeno, playerID, 4), model.CreditsFromUnits(2))); err != nil {
		t.Fatalf("la orden ajena devolvió %v", err)
	}

	orders, err := fixture.market.ListOpenOrders(t.Context(), propio, nil)
	if err != nil {
		t.Fatalf("ListOpenOrders devolvió %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("se devolvieron %d órdenes, se esperaban las 2 propias", len(orders))
	}
	for _, order := range orders {
		if order.UserID != propio {
			t.Errorf("se devolvió una orden del usuario %d: el listado es solo del que pregunta", order.UserID)
		}
	}

	filtradas, err := fixture.market.ListOpenOrders(t.Context(), propio, &playerID)
	if err != nil {
		t.Fatalf("ListOpenOrders con filtro devolvió %v", err)
	}
	if len(filtradas) != 1 || filtradas[0].PlayerID != playerID {
		t.Errorf("el filtro por jugador devolvió %d órdenes, se esperaba solo la de ese jugador", len(filtradas))
	}

	if _, err = fixture.market.CancelOrder(t.Context(), cancelCommand(propio, propia.Order.ID)); err != nil {
		t.Fatalf("CancelOrder devolvió %v", err)
	}

	restantes, err := fixture.market.ListOpenOrders(t.Context(), propio, nil)
	if err != nil {
		t.Fatalf("ListOpenOrders tras cancelar devolvió %v", err)
	}
	if len(restantes) != 1 {
		t.Errorf("quedaron %d órdenes vivas, se esperaba 1: la cancelada no se lista", len(restantes))
	}
}
