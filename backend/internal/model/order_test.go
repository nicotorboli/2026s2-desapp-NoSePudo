package model_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var instante = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

func ordenDe(status model.OrderStatus, quantity, executed int64) model.Order {
	return model.Order{
		Side:             model.SideBuy,
		TimeInForce:      model.GoodTillCancelled,
		Status:           status,
		Quantity:         quantity,
		ExecutedQuantity: executed,
		Price:            model.CreditsFromUnits(1),
	}
}

func TestOrderRemainingQuantityDescuentaLoEjecutado(t *testing.T) {
	order := ordenDe(model.StatusPartiallyFilled, 10, 4)

	if got := order.RemainingQuantity(); got != 6 {
		t.Errorf("RemainingQuantity() = %d, se esperaba 6", got)
	}
}

func TestOrderIsLiveSoloEnLosDosEstadosVivos(t *testing.T) {
	cases := []struct {
		status model.OrderStatus
		want   bool
	}{
		{status: model.StatusPending, want: true},
		{status: model.StatusPartiallyFilled, want: true},
		{status: model.StatusFilled, want: false},
		{status: model.StatusCancelled, want: false},
	}

	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			order := ordenDe(c.status, 10, 0)
			if got := order.IsLive(); got != c.want {
				t.Errorf("IsLive() = %v, se esperaba %v", got, c.want)
			}
		})
	}
}

func TestOrderExecuteTransicionesValidas(t *testing.T) {
	cases := []struct {
		name       string
		status     model.OrderStatus
		wantStatus model.OrderStatus
		quantity   int64
		executed   int64
		execute    int64
	}{
		{name: "pendiente ejecutada en parte queda parcial", status: model.StatusPending, quantity: 10, executed: 0, execute: 4, wantStatus: model.StatusPartiallyFilled},
		{name: "pendiente ejecutada entera queda completa", status: model.StatusPending, quantity: 10, executed: 0, execute: 10, wantStatus: model.StatusFilled},
		{name: "parcial que completa el resto queda completa", status: model.StatusPartiallyFilled, quantity: 10, executed: 6, execute: 4, wantStatus: model.StatusFilled},
		{name: "parcial que ejecuta menos que el resto sigue parcial", status: model.StatusPartiallyFilled, quantity: 10, executed: 4, execute: 2, wantStatus: model.StatusPartiallyFilled},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			order := ordenDe(c.status, c.quantity, c.executed)

			if err := order.Execute(c.execute, instante); err != nil {
				t.Fatalf("Execute devolvió %v, se esperaba nil", err)
			}
			if order.Status != c.wantStatus {
				t.Errorf("Status = %s, se esperaba %s", order.Status, c.wantStatus)
			}
			if order.ExecutedQuantity != c.executed+c.execute {
				t.Errorf("ExecutedQuantity = %d, se esperaba %d", order.ExecutedQuantity, c.executed+c.execute)
			}
			if !order.UpdatedAt.Equal(instante) {
				t.Errorf("UpdatedAt = %v, se esperaba el instante recibido", order.UpdatedAt)
			}
		})
	}
}

func TestOrderExecuteTransicionesInvalidas(t *testing.T) {
	cases := []struct {
		wantErr  error
		name     string
		status   model.OrderStatus
		quantity int64
		executed int64
		execute  int64
	}{
		{name: "una orden completa no ejecuta mas", status: model.StatusFilled, quantity: 10, executed: 10, execute: 1, wantErr: model.ErrOrderNotLive},
		{name: "una orden cancelada no ejecuta", status: model.StatusCancelled, quantity: 10, executed: 0, execute: 1, wantErr: model.ErrOrderNotLive},
		{name: "cantidad cero", status: model.StatusPending, quantity: 10, executed: 0, execute: 0, wantErr: model.ErrInvalidQuantity},
		{name: "cantidad negativa", status: model.StatusPending, quantity: 10, executed: 0, execute: -1, wantErr: model.ErrInvalidQuantity},
		{name: "mas que el remanente", status: model.StatusPartiallyFilled, quantity: 10, executed: 6, execute: 5, wantErr: model.ErrInvalidQuantity},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			order := ordenDe(c.status, c.quantity, c.executed)
			before := order

			err := order.Execute(c.execute, instante)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("Execute devolvió %v, se esperaba %v", err, c.wantErr)
			}
			if order != before {
				t.Error("una transición rechazada dejó la orden modificada")
			}
		})
	}
}

func TestOrderCancelDesdeLosEstadosVivos(t *testing.T) {
	for _, status := range []model.OrderStatus{model.StatusPending, model.StatusPartiallyFilled} {
		t.Run(string(status), func(t *testing.T) {
			order := ordenDe(status, 10, 2)

			if err := order.Cancel(instante); err != nil {
				t.Fatalf("Cancel devolvió %v, se esperaba nil", err)
			}
			if order.Status != model.StatusCancelled {
				t.Errorf("Status = %s, se esperaba CANCELLED", order.Status)
			}
			if order.ExecutedQuantity != 2 {
				t.Errorf("cancelar movió lo ya ejecutado: %d", order.ExecutedQuantity)
			}
		})
	}
}

func TestOrderCancelDesdeUnEstadoFinalEsRechazada(t *testing.T) {
	for _, status := range []model.OrderStatus{model.StatusFilled, model.StatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			order := ordenDe(status, 10, 10)

			if err := order.Cancel(instante); !errors.Is(err, model.ErrOrderNotLive) {
				t.Errorf("Cancel devolvió %v, se esperaba ErrOrderNotLive", err)
			}
		})
	}
}

func TestOppositeSideDevuelveElLadoContrario(t *testing.T) {
	if got := model.OppositeSide(model.SideBuy); got != model.SideSell {
		t.Errorf("OppositeSide(BUY) = %s, se esperaba SELL", got)
	}
	if got := model.OppositeSide(model.SideSell); got != model.SideBuy {
		t.Errorf("OppositeSide(SELL) = %s, se esperaba BUY", got)
	}
}
