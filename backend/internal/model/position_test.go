package model_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func posicionCon(available, reserved int64, costBasis model.Credits) model.Position {
	return model.Position{
		UserID:            1,
		PlayerID:          7,
		AvailableQuantity: available,
		ReservedQuantity:  reserved,
		CostBasis:         costBasis,
	}
}

func TestPositionTotalQuantitySumaDisponibleYReservado(t *testing.T) {
	position := posicionCon(6, 4, model.CreditsFromUnits(10))

	if got := position.TotalQuantity(); got != 10 {
		t.Errorf("TotalQuantity() = %d, se esperaba 10", got)
	}
}

func TestPositionAveragePrice(t *testing.T) {
	cases := []struct {
		name      string
		available int64
		reserved  int64
		costBasis model.Credits
		want      model.Credits
	}{
		{name: "diez tokens a un credito", available: 10, reserved: 0, costBasis: model.CreditsFromUnits(10), want: model.CreditsFromUnits(1)},
		{name: "compras mezcladas a uno y a dos", available: 20, reserved: 0, costBasis: model.CreditsFromUnits(30), want: model.CreditsFromUnits(1) + 5000},
		{name: "posicion vacia no divide por cero", available: 0, reserved: 0, costBasis: 0, want: 0},
		{name: "los reservados cuentan para el promedio", available: 5, reserved: 5, costBasis: model.CreditsFromUnits(10), want: model.CreditsFromUnits(1)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			position := posicionCon(c.available, c.reserved, c.costBasis)
			if got := position.AveragePrice(); got != c.want {
				t.Errorf("AveragePrice() = %d, se esperaba %d", got, c.want)
			}
		})
	}
}

func TestPositionReserveTokensMueveDeDisponibleAReservado(t *testing.T) {
	position := posicionCon(10, 0, model.CreditsFromUnits(10))

	if err := position.ReserveTokens(4); err != nil {
		t.Fatalf("ReserveTokens devolvió %v, se esperaba nil", err)
	}
	if position.AvailableQuantity != 6 || position.ReservedQuantity != 4 {
		t.Errorf("disponible = %d y reservado = %d, se esperaba 6 y 4", position.AvailableQuantity, position.ReservedQuantity)
	}
	if position.CostBasis != model.CreditsFromUnits(10) {
		t.Errorf("reservar movió el costo: %d", position.CostBasis)
	}
}

func TestPositionReserveTokensRechazos(t *testing.T) {
	cases := []struct {
		wantErr  error
		name     string
		quantity int64
	}{
		{name: "cantidad cero", quantity: 0, wantErr: model.ErrInvalidQuantity},
		{name: "cantidad negativa", quantity: -1, wantErr: model.ErrInvalidQuantity},
		{name: "un token mas de los disponibles", quantity: 7, wantErr: model.ErrInsufficientTokens},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			position := posicionCon(6, 4, model.CreditsFromUnits(10))
			before := position

			if err := position.ReserveTokens(c.quantity); !errors.Is(err, c.wantErr) {
				t.Fatalf("ReserveTokens devolvió %v, se esperaba %v", err, c.wantErr)
			}
			if position != before {
				t.Error("una reserva rechazada dejó la posición modificada")
			}
		})
	}
}

func TestPositionReleaseReservedTokensDevuelveLoReservado(t *testing.T) {
	position := posicionCon(6, 4, model.CreditsFromUnits(10))

	if err := position.ReleaseReservedTokens(4); err != nil {
		t.Fatalf("ReleaseReservedTokens devolvió %v, se esperaba nil", err)
	}
	if position.AvailableQuantity != 10 || position.ReservedQuantity != 0 {
		t.Errorf("disponible = %d y reservado = %d, se esperaba 10 y 0", position.AvailableQuantity, position.ReservedQuantity)
	}
}

func TestPositionReleaseReservedTokensRechazaMasDeLoReservado(t *testing.T) {
	position := posicionCon(6, 4, model.CreditsFromUnits(10))

	if err := position.ReleaseReservedTokens(5); !errors.Is(err, model.ErrReservationTooSmall) {
		t.Errorf("ReleaseReservedTokens devolvió %v, se esperaba ErrReservationTooSmall", err)
	}
}

func TestPositionDeliverReservedTokensLiberaElCostoEnProporcion(t *testing.T) {
	position := posicionCon(6, 4, model.CreditsFromUnits(10))

	released, err := position.DeliverReservedTokens(4)
	if err != nil {
		t.Fatalf("DeliverReservedTokens devolvió %v, se esperaba nil", err)
	}
	if released != model.CreditsFromUnits(4) {
		t.Errorf("costo liberado = %d, se esperaban 4 créditos", released)
	}
	if position.TotalQuantity() != 6 {
		t.Errorf("TotalQuantity() = %d, se esperaba 6", position.TotalQuantity())
	}
	if position.CostBasis != model.CreditsFromUnits(6) {
		t.Errorf("CostBasis = %d, se esperaban 6 créditos", position.CostBasis)
	}
}

func TestPositionDeliverReservedTokensDejaElPrecioPromedioIntacto(t *testing.T) {
	position := posicionCon(6, 4, model.CreditsFromUnits(10))
	averageBefore := position.AveragePrice()

	if _, err := position.DeliverReservedTokens(4); err != nil {
		t.Fatalf("DeliverReservedTokens devolvió %v, se esperaba nil", err)
	}

	if position.AveragePrice() != averageBefore {
		t.Errorf("el precio promedio pasó de %d a %d: una venta no lo cambia",
			averageBefore, position.AveragePrice())
	}
}

func TestPositionDeliverReservedTokensDeLaPosicionEnteraNoDejaResiduo(t *testing.T) {
	position := posicionCon(0, 3, model.CreditsFromUnits(10))

	released, err := position.DeliverReservedTokens(3)
	if err != nil {
		t.Fatalf("DeliverReservedTokens devolvió %v, se esperaba nil", err)
	}
	if released != model.CreditsFromUnits(10) {
		t.Errorf("costo liberado = %d, se esperaba el costo entero de 10 créditos", released)
	}
	if position.CostBasis != 0 {
		t.Errorf("CostBasis = %d: una posición sin tokens no puede conservar costo", position.CostBasis)
	}
	if position.TotalQuantity() != 0 {
		t.Errorf("TotalQuantity() = %d, se esperaba 0", position.TotalQuantity())
	}
}

func TestPositionDeliverReservedTokensRechazos(t *testing.T) {
	cases := []struct {
		wantErr  error
		name     string
		quantity int64
	}{
		{name: "cantidad cero", quantity: 0, wantErr: model.ErrInvalidQuantity},
		{name: "cantidad negativa", quantity: -2, wantErr: model.ErrInvalidQuantity},
		{name: "mas de lo reservado", quantity: 5, wantErr: model.ErrReservationTooSmall},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			position := posicionCon(6, 4, model.CreditsFromUnits(10))
			before := position

			released, err := position.DeliverReservedTokens(c.quantity)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("DeliverReservedTokens devolvió %v, se esperaba %v", err, c.wantErr)
			}
			if released != 0 {
				t.Errorf("una entrega rechazada liberó %d de costo", released)
			}
			if position != before {
				t.Error("una entrega rechazada dejó la posición modificada")
			}
		})
	}
}

func TestPositionReceiveSumaTokensYCosto(t *testing.T) {
	position := posicionCon(10, 0, model.CreditsFromUnits(10))

	if err := position.Receive(10, model.CreditsFromUnits(20)); err != nil {
		t.Fatalf("Receive devolvió %v, se esperaba nil", err)
	}
	if position.AvailableQuantity != 20 {
		t.Errorf("AvailableQuantity = %d, se esperaba 20", position.AvailableQuantity)
	}
	if position.AveragePrice() != model.CreditsFromUnits(1)+5000 {
		t.Errorf("AveragePrice() = %d, se esperaba 1.5 créditos", position.AveragePrice())
	}
}

func TestPositionReceiveRechazaCantidadOCostoNoPositivos(t *testing.T) {
	cases := []struct {
		name     string
		quantity int64
		cost     model.Credits
	}{
		{name: "cantidad cero", quantity: 0, cost: model.CreditsFromUnits(1)},
		{name: "cantidad negativa", quantity: -1, cost: model.CreditsFromUnits(1)},
		{name: "costo cero", quantity: 1, cost: 0},
		{name: "costo negativo", quantity: 1, cost: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			position := posicionCon(10, 0, model.CreditsFromUnits(10))

			if err := position.Receive(c.quantity, c.cost); !errors.Is(err, model.ErrInvalidQuantity) {
				t.Errorf("Receive devolvió %v, se esperaba ErrInvalidQuantity", err)
			}
		})
	}
}
