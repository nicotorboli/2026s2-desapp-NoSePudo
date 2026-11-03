package model_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestCreditsFromUnitsEscalaAUnaDiezmilesima(t *testing.T) {
	cases := []struct {
		name  string
		units int64
		want  model.Credits
	}{
		{"cero", 0, 0},
		{"un credito", 1, 10000},
		{"el saldo inicial por defecto", 1000, 10000000},
		{"negativo", -3, -30000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := model.CreditsFromUnits(c.units); got != c.want {
				t.Errorf("CreditsFromUnits(%d) = %d, se esperaba %d", c.units, got, c.want)
			}
		})
	}
}

func TestCreditsTimesMultiplicaSinPerderPrecision(t *testing.T) {
	price := model.Credits(15000)

	if got := price.Times(3); got != 45000 {
		t.Errorf("Times(3) = %d, se esperaba 45000", got)
	}
	if got := price.Times(0); got != 0 {
		t.Errorf("Times(0) = %d, se esperaba 0", got)
	}
}

func TestCreditsPlusYMinusSonInversas(t *testing.T) {
	balance := model.CreditsFromUnits(1000)
	amount := model.Credits(12345)

	if got := balance.Plus(amount).Minus(amount); got != balance {
		t.Errorf("Plus seguido de Minus = %d, se esperaba %d", got, balance)
	}
}

func TestCreditsDividedByTruncaHaciaCero(t *testing.T) {
	cases := []struct {
		name     string
		total    model.Credits
		quantity int64
		want     model.Credits
	}{
		{"division exacta", 30000, 3, 10000},
		{"trunca el resto positivo", 10000, 3, 3333},
		{"trunca el resto negativo", -10000, 3, -3333},
		{"dividir por cero devuelve cero en vez de entrar en panico", 10000, 0, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.total.DividedBy(c.quantity); got != c.want {
				t.Errorf("DividedBy(%d) = %d, se esperaba %d", c.quantity, got, c.want)
			}
		})
	}
}

func TestCreditsPreservanLaPrecisionDeCuatroDecimales(t *testing.T) {
	oneTenThousandth := model.Credits(1)

	if got := oneTenThousandth.Times(10000); got != model.CreditsFromUnits(1) {
		t.Errorf("diez mil veces una diezmilesima = %d, se esperaba un credito entero", got)
	}
}
