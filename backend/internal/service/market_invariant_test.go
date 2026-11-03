package service_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func (f *marketFixture) reservedCredits(t *testing.T) int64 {
	t.Helper()

	var total int64
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COALESCE(SUM(reserved_balance), 0) FROM user_accounts`,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudo sumar los créditos reservados: %v", err)
	}

	return total
}

func (f *marketFixture) creditsCommittedByLiveOrders(t *testing.T) int64 {
	t.Helper()

	var total int64
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COALESCE(SUM(price * (quantity - executed_quantity)), 0)
		 FROM market_orders
		 WHERE side = 'BUY' AND status IN ('PENDING', 'PARTIALLY_FILLED')`,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudo sumar lo comprometido por las órdenes vivas: %v", err)
	}

	return total
}

func (f *marketFixture) reservedTokens(t *testing.T) int64 {
	t.Helper()

	var total int64
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COALESCE(SUM(reserved_quantity), 0) FROM user_positions`,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudo sumar los tokens reservados: %v", err)
	}

	return total
}

func (f *marketFixture) tokensCommittedByLiveOrders(t *testing.T) int64 {
	t.Helper()

	var total int64
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COALESCE(SUM(quantity - executed_quantity), 0)
		 FROM market_orders
		 WHERE side = 'SELL' AND status IN ('PENDING', 'PARTIALLY_FILLED')`,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudieron sumar los tokens comprometidos por las órdenes vivas: %v", err)
	}

	return total
}

func (f *marketFixture) tokensOfPlayer(t *testing.T, playerID int64) int64 {
	t.Helper()

	var total int64
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COALESCE(SUM(available_quantity + reserved_quantity), 0)
		 FROM user_positions WHERE player_id = $1`, playerID,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudieron sumar los tokens del jugador %d: %v", playerID, err)
	}

	return total
}

func (f *marketFixture) assertNoReservationOutlivesItsOrder(t *testing.T, momento string) {
	t.Helper()

	if reserved, committed := f.reservedCredits(t), f.creditsCommittedByLiveOrders(t); reserved != committed {
		t.Errorf("%s: hay %d créditos reservados y las órdenes vivas comprometen %d", momento, reserved, committed)
	}
	if reserved, committed := f.reservedTokens(t), f.tokensCommittedByLiveOrders(t); reserved != committed {
		t.Errorf("%s: hay %d tokens reservados y las órdenes vivas comprometen %d", momento, reserved, committed)
	}
	if !f.ledgerIsBalanced(t) {
		t.Errorf("%s: hay una liquidación del libro mayor que no suma cero", momento)
	}
}

func TestNingunaReservaSobreviveASuOrden(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	pobre := fixture.seedUser(t, model.CreditsFromUnits(1))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 5)

	fixture.assertNoReservationOutlivesItsOrder(t, "al empezar")

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(pobre, playerID, 100)); !errors.Is(err, model.ErrInsufficientFunds) {
		t.Fatalf("la compra sin fondos devolvió %v", err)
	}
	fixture.assertNoReservationOutlivesItsOrder(t, "tras una compra rechazada por fondos")

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 50)); !errors.Is(err, service.ErrNoLiquidity) {
		t.Fatalf("el FOK sin liquidez devolvió %v", err)
	}
	fixture.assertNoReservationOutlivesItsOrder(t, "tras un FOK deshecho")

	conIOC := withTimeInForce(
		withPrice(buyCommand(buyer, playerID, 8), model.CreditsFromUnits(2)),
		model.ImmediateOrCancel,
	)
	if _, err := fixture.market.PlaceOrder(t.Context(), conIOC); err != nil {
		t.Fatalf("el IOC devolvió %v", err)
	}
	fixture.assertNoReservationOutlivesItsOrder(t, "tras un IOC con remanente cancelado")

	enReposo, err := fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(buyer, playerID, 6), model.CreditsFromUnits(3)))
	if err != nil {
		t.Fatalf("la orden en reposo devolvió %v", err)
	}
	fixture.assertNoReservationOutlivesItsOrder(t, "con una orden viva en reposo")

	if _, err = fixture.market.CancelOrder(t.Context(), cancelCommand(buyer, enReposo.Order.ID)); err != nil {
		t.Fatalf("CancelOrder devolvió %v", err)
	}
	fixture.assertNoReservationOutlivesItsOrder(t, "tras cancelar la orden en reposo")
}

func TestElRemanenteEnReposoCruzaConUnaOrdenPosterior(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))

	enReposo, err := fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(buyer, playerID, 6), model.CreditsFromUnits(3)))
	if err != nil {
		t.Fatalf("la compra en reposo devolvió %v", err)
	}
	if enReposo.Order.Status != model.StatusPending {
		t.Fatalf("la compra quedó en %s, se esperaba PENDING para que espere una venta", enReposo.Order.Status)
	}

	venta, err := fixture.market.PlaceOrder(t.Context(), withPrice(sellCommand(seller, playerID, 4), model.CreditsFromUnits(1)))
	if err != nil {
		t.Fatalf("la venta posterior devolvió %v", err)
	}

	if venta.Order.Status != model.StatusFilled {
		t.Errorf("la venta quedó en %s, se esperaba FILLED contra el remanente en reposo", venta.Order.Status)
	}
	if len(venta.Trades) != 1 {
		t.Fatalf("se registraron %d operaciones, se esperaba 1", len(venta.Trades))
	}
	if venta.Trades[0].Price != model.CreditsFromUnits(3) {
		t.Errorf("la ejecución fue a %d, se esperaba el precio de la compra en reposo", venta.Trades[0].Price)
	}

	if fixture.orderStatus(t, enReposo.Order.ID) != model.StatusPartiallyFilled {
		t.Error("la compra en reposo no quedó PARTIALLY_FILLED tras ser cruzada")
	}
	if total := fixture.tokensOfPlayer(t, playerID); total != 10 {
		t.Errorf("hay %d tokens del jugador, se esperaban los 10 sembrados: una operación los mueve, no los crea", total)
	}

	fixture.assertNoReservationOutlivesItsOrder(t, "tras cruzar el remanente")
}
