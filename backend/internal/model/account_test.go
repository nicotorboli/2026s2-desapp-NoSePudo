package model_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func cuentaCon(available, reserved model.Credits) model.Account {
	return model.Account{UserID: 1, AvailableBalance: available, ReservedBalance: reserved}
}

func TestAccountReserveMueveDeDisponibleAReservado(t *testing.T) {
	account := cuentaCon(model.CreditsFromUnits(1000), 0)

	if err := account.Reserve(model.CreditsFromUnits(10)); err != nil {
		t.Fatalf("Reserve devolvió %v, se esperaba nil", err)
	}
	if account.AvailableBalance != model.CreditsFromUnits(990) {
		t.Errorf("AvailableBalance = %d, se esperaba 990 créditos", account.AvailableBalance)
	}
	if account.ReservedBalance != model.CreditsFromUnits(10) {
		t.Errorf("ReservedBalance = %d, se esperaba 10 créditos", account.ReservedBalance)
	}
}

func TestAccountReserveHastaElUltimoCredito(t *testing.T) {
	account := cuentaCon(model.CreditsFromUnits(10), 0)

	if err := account.Reserve(model.CreditsFromUnits(10)); err != nil {
		t.Fatalf("Reserve del saldo entero devolvió %v", err)
	}
	if account.AvailableBalance != 0 {
		t.Errorf("AvailableBalance = %d, se esperaba 0", account.AvailableBalance)
	}
}

func TestAccountReserveRechazos(t *testing.T) {
	cases := []struct {
		wantErr   error
		name      string
		available model.Credits
		amount    model.Credits
	}{
		{name: "monto cero", available: model.CreditsFromUnits(1000), amount: 0, wantErr: model.ErrInvalidAmount},
		{name: "monto negativo", available: model.CreditsFromUnits(1000), amount: -1, wantErr: model.ErrInvalidAmount},
		{name: "una diezmilésima por encima del saldo", available: model.CreditsFromUnits(10), amount: model.CreditsFromUnits(10) + 1, wantErr: model.ErrInsufficientFunds},
		{name: "saldo vacío", available: 0, amount: 1, wantErr: model.ErrInsufficientFunds},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			account := cuentaCon(c.available, 0)
			before := account

			if err := account.Reserve(c.amount); !errors.Is(err, c.wantErr) {
				t.Fatalf("Reserve devolvió %v, se esperaba %v", err, c.wantErr)
			}
			if account != before {
				t.Error("una reserva rechazada dejó la cuenta modificada")
			}
		})
	}
}

func TestAccountReleaseReservationDevuelveLoReservado(t *testing.T) {
	account := cuentaCon(model.CreditsFromUnits(990), model.CreditsFromUnits(10))

	if err := account.ReleaseReservation(model.CreditsFromUnits(10)); err != nil {
		t.Fatalf("ReleaseReservation devolvió %v, se esperaba nil", err)
	}
	if account.AvailableBalance != model.CreditsFromUnits(1000) {
		t.Errorf("AvailableBalance = %d, se esperaba 1000 créditos", account.AvailableBalance)
	}
	if account.ReservedBalance != 0 {
		t.Errorf("ReservedBalance = %d, se esperaba 0", account.ReservedBalance)
	}
}

func TestAccountSettleReservationConsumeSinAcreditar(t *testing.T) {
	account := cuentaCon(model.CreditsFromUnits(990), model.CreditsFromUnits(10))

	if err := account.SettleReservation(model.CreditsFromUnits(10)); err != nil {
		t.Fatalf("SettleReservation devolvió %v, se esperaba nil", err)
	}
	if account.AvailableBalance != model.CreditsFromUnits(990) {
		t.Errorf("AvailableBalance = %d, liquidar no debe acreditar", account.AvailableBalance)
	}
	if account.ReservedBalance != 0 {
		t.Errorf("ReservedBalance = %d, se esperaba 0", account.ReservedBalance)
	}
}

func TestAccountLiberarYLiquidarRechazanMasDeLoReservado(t *testing.T) {
	cases := []struct {
		wantErr error
		apply   func(*model.Account, model.Credits) error
		name    string
		amount  model.Credits
	}{
		{name: "liberar cero", apply: (*model.Account).ReleaseReservation, amount: 0, wantErr: model.ErrInvalidAmount},
		{name: "liberar más de lo reservado", apply: (*model.Account).ReleaseReservation, amount: model.CreditsFromUnits(11), wantErr: model.ErrReservationTooSmall},
		{name: "liquidar cero", apply: (*model.Account).SettleReservation, amount: 0, wantErr: model.ErrInvalidAmount},
		{name: "liquidar más de lo reservado", apply: (*model.Account).SettleReservation, amount: model.CreditsFromUnits(11), wantErr: model.ErrReservationTooSmall},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			account := cuentaCon(model.CreditsFromUnits(990), model.CreditsFromUnits(10))
			before := account

			if err := c.apply(&account, c.amount); !errors.Is(err, c.wantErr) {
				t.Fatalf("devolvió %v, se esperaba %v", err, c.wantErr)
			}
			if account != before {
				t.Error("una operación rechazada dejó la cuenta modificada")
			}
		})
	}
}

func TestAccountCreditSumaAlDisponible(t *testing.T) {
	account := cuentaCon(model.CreditsFromUnits(100), 0)

	if err := account.Credit(model.CreditsFromUnits(4)); err != nil {
		t.Fatalf("Credit devolvió %v, se esperaba nil", err)
	}
	if account.AvailableBalance != model.CreditsFromUnits(104) {
		t.Errorf("AvailableBalance = %d, se esperaba 104 créditos", account.AvailableBalance)
	}
}

func TestAccountCreditRechazaMontosNoPositivos(t *testing.T) {
	for _, amount := range []model.Credits{0, -1} {
		account := cuentaCon(model.CreditsFromUnits(100), 0)

		if err := account.Credit(amount); !errors.Is(err, model.ErrInvalidAmount) {
			t.Errorf("Credit(%d) devolvió %v, se esperaba ErrInvalidAmount", amount, err)
		}
	}
}
