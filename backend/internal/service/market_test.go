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
