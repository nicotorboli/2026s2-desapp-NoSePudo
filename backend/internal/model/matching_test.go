package model_test

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const (
	comprador = int64(1)
	vendedor  = int64(2)
	jugador   = int64(7)
)

func restingOrder(id int64, userID int64, side model.OrderSide, price model.Credits, quantity int64, minutesOld int) model.Order {
	return model.Order{
		CreatedAt:   instante.Add(-time.Duration(minutesOld) * time.Minute),
		Side:        side,
		TimeInForce: model.GoodTillCancelled,
		Status:      model.StatusPending,
		ID:          id,
		UserID:      userID,
		PlayerID:    jugador,
		Quantity:    quantity,
		Price:       price,
	}
}

func incomingOrder(side model.OrderSide, price model.Credits, quantity int64) model.Order {
	return model.Order{
		CreatedAt:   instante,
		Side:        side,
		TimeInForce: model.GoodTillCancelled,
		Status:      model.StatusPending,
		ID:          99,
		UserID:      comprador,
		PlayerID:    jugador,
		Quantity:    quantity,
		Price:       price,
	}
}

func assertExecutions(t *testing.T, got []model.Execution, want []model.Execution) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("se obtuvieron %d ejecuciones y se esperaban %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ejecución %d = %+v, se esperaba %+v", i, got[i], want[i])
		}
	}
}

func TestMatchOrderSoloCruzaContraElLadoContrarioDelMismoJugador(t *testing.T) {
	otroJugador := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)
	otroJugador.PlayerID = jugador + 1
	mismoLado := restingOrder(2, vendedor, model.SideBuy, model.CreditsFromUnits(1), 10, 5)
	elegible := restingOrder(3, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 10),
		[]model.Order{otroJugador, mismoLado, elegible},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 3, Quantity: 10, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderIgnoraLosPreciosQueNoCruzan(t *testing.T) {
	caro := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(3), 10, 5)
	alLimite := restingOrder(2, vendedor, model.SideSell, model.CreditsFromUnits(2), 10, 5)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(2), 20),
		[]model.Order{caro, alLimite},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 10, Price: model.CreditsFromUnits(2)},
	})
}

func TestMatchOrderVendeContraLosPreciosDeCompraQueCruzan(t *testing.T) {
	barato := restingOrder(1, vendedor, model.SideBuy, model.CreditsFromUnits(1), 10, 5)
	alLimite := restingOrder(2, vendedor, model.SideBuy, model.CreditsFromUnits(2), 10, 5)

	executions := model.MatchOrder(
		incomingOrder(model.SideSell, model.CreditsFromUnits(2), 20),
		[]model.Order{barato, alLimite},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 10, Price: model.CreditsFromUnits(2)},
	})
}

func TestMatchOrderUnaCompraConsumePrimeroLoMasBarato(t *testing.T) {
	caro := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(3), 5, 10)
	barato := restingOrder(2, vendedor, model.SideSell, model.CreditsFromUnits(1), 5, 1)
	medio := restingOrder(3, vendedor, model.SideSell, model.CreditsFromUnits(2), 5, 1)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(3), 15),
		[]model.Order{caro, barato, medio},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 5, Price: model.CreditsFromUnits(1)},
		{RestingOrderID: 3, Quantity: 5, Price: model.CreditsFromUnits(2)},
		{RestingOrderID: 1, Quantity: 5, Price: model.CreditsFromUnits(3)},
	})
}

func TestMatchOrderUnaVentaConsumePrimeroLoMasCaro(t *testing.T) {
	barato := restingOrder(1, vendedor, model.SideBuy, model.CreditsFromUnits(1), 5, 10)
	caro := restingOrder(2, vendedor, model.SideBuy, model.CreditsFromUnits(3), 1, 1)

	executions := model.MatchOrder(
		incomingOrder(model.SideSell, model.CreditsFromUnits(1), 6),
		[]model.Order{barato, caro},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 1, Price: model.CreditsFromUnits(3)},
		{RestingOrderID: 1, Quantity: 5, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderAIgualPrecioConsumePrimeroLaMasVieja(t *testing.T) {
	nueva := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 5, 1)
	vieja := restingOrder(2, vendedor, model.SideSell, model.CreditsFromUnits(1), 5, 30)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 10),
		[]model.Order{nueva, vieja},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 5, Price: model.CreditsFromUnits(1)},
		{RestingOrderID: 1, Quantity: 5, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderSalteaLasOrdenesDelMismoUsuario(t *testing.T) {
	propia := restingOrder(1, comprador, model.SideSell, model.CreditsFromUnits(1), 10, 30)
	ajena := restingOrder(2, vendedor, model.SideSell, model.CreditsFromUnits(2), 10, 1)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(2), 10),
		[]model.Order{propia, ajena},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 2, Quantity: 10, Price: model.CreditsFromUnits(2)},
	})
}

func TestMatchOrderEjecutaAlPrecioDeLaOrdenQueDescansa(t *testing.T) {
	barata := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(5), 10),
		[]model.Order{barata},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 1, Quantity: 10, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderCadaOrdenAportaComoMuchoSuRemanente(t *testing.T) {
	parcial := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)
	parcial.ExecutedQuantity = 7
	parcial.Status = model.StatusPartiallyFilled

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 10),
		[]model.Order{parcial},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 1, Quantity: 3, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderCortaCuandoElRemanenteLlegaACero(t *testing.T) {
	primera := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 30)
	segunda := restingOrder(2, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 1)

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 4),
		[]model.Order{primera, segunda},
	)

	assertExecutions(t, executions, []model.Execution{
		{RestingOrderID: 1, Quantity: 4, Price: model.CreditsFromUnits(1)},
	})
}

func TestMatchOrderSinLibroNoEjecutaNada(t *testing.T) {
	executions := model.MatchOrder(incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 10), nil)

	if len(executions) != 0 {
		t.Errorf("se obtuvieron %d ejecuciones contra un libro vacío", len(executions))
	}
}

func TestMatchOrderNoMutaSusEntradas(t *testing.T) {
	resting := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)
	book := []model.Order{resting}
	incoming := incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 4)
	incomingBefore := incoming

	model.MatchOrder(incoming, book)

	if book[0] != resting {
		t.Error("el matcher modificó una orden del libro")
	}
	if incoming != incomingBefore {
		t.Error("el matcher modificó la orden entrante")
	}
}

func TestMatchOrderIgnoraLasOrdenesQueYaNoEstanVivas(t *testing.T) {
	cancelada := restingOrder(1, vendedor, model.SideSell, model.CreditsFromUnits(1), 10, 5)
	cancelada.Status = model.StatusCancelled

	executions := model.MatchOrder(
		incomingOrder(model.SideBuy, model.CreditsFromUnits(1), 10),
		[]model.Order{cancelada},
	)

	if len(executions) != 0 {
		t.Errorf("se cruzó contra una orden cancelada: %+v", executions)
	}
}
