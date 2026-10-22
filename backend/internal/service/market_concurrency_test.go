package service_test

import (
	"sync"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func placeConcurrently(t *testing.T, fixture *marketFixture, commands []service.PlaceOrderCommand) []error {
	t.Helper()

	errs := make([]error, len(commands))

	var waiting sync.WaitGroup
	var ready sync.WaitGroup

	ready.Add(len(commands))
	start := make(chan struct{})

	for i := range commands {
		waiting.Add(1)

		go func(index int) {
			defer waiting.Done()

			ready.Done()
			<-start

			_, errs[index] = fixture.market.PlaceOrder(t.Context(), commands[index])
		}(i)
	}

	ready.Wait()
	close(start)
	waiting.Wait()

	return errs
}

func countNil(errs []error) int {
	total := 0
	for _, err := range errs {
		if err == nil {
			total++
		}
	}

	return total
}

func TestCompradoresConcurrentesNoSobrevendenLosUltimosTokens(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.db.SetMaxOpenConns(20)

	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 5)

	const compradores = 10

	commands := make([]service.PlaceOrderCommand, compradores)
	for i := range commands {
		buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
		commands[i] = buyCommand(buyer, playerID, 1)
	}

	errs := placeConcurrently(t, fixture, commands)

	if exitosas := countNil(errs); exitosas != 5 {
		t.Errorf("se completaron %d compras de un token, se esperaban exactamente 5: hay 5 tokens ofrecidos", exitosas)
	}

	if total := fixture.tokensOfPlayer(t, playerID); total != 5 {
		t.Errorf("hay %d tokens del jugador, se esperaban los 5 sembrados: ninguna operación los crea ni los destruye", total)
	}

	position := fixture.position(t, seller, playerID)
	if position.ReservedQuantity != 0 || position.AvailableQuantity != 0 {
		t.Errorf("al vendedor le quedaron %d disponibles y %d reservados, se esperaba cero y cero",
			position.AvailableQuantity, position.ReservedQuantity)
	}

	fixture.assertNoReservationOutlivesItsOrder(t, "tras diez compras concurrentes")
}

func TestOrdenesConcurrentesDeUnUsuarioNoSobregiranSuSaldo(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.db.SetMaxOpenConns(20)

	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, 0)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(5))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(1))
	fixture.seedRestingSell(t, seller, playerID, model.CreditsFromUnits(1), 100)

	const intentos = 10

	commands := make([]service.PlaceOrderCommand, intentos)
	for i := range commands {
		commands[i] = buyCommand(buyer, playerID, 1)
	}

	errs := placeConcurrently(t, fixture, commands)

	if exitosas := countNil(errs); exitosas != 5 {
		t.Errorf("se completaron %d compras, se esperaban exactamente 5: el saldo alcanza para 5 tokens a 1 crédito", exitosas)
	}

	account := fixture.account(t, buyer)
	if account.AvailableBalance != 0 {
		t.Errorf("al comprador le quedaron %d créditos, se esperaba cero", account.AvailableBalance)
	}
	if account.ReservedBalance != 0 {
		t.Errorf("al comprador le quedaron %d créditos reservados, se esperaba cero", account.ReservedBalance)
	}

	position := fixture.position(t, buyer, playerID)
	if position.AvailableQuantity != 5 {
		t.Errorf("el comprador quedó con %d tokens, se esperaban 5", position.AvailableQuantity)
	}

	fixture.assertNoReservationOutlivesItsOrder(t, "tras diez órdenes concurrentes de un mismo usuario")
}
