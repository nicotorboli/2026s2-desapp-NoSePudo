package model

import "errors"

var (
	ErrInvalidAmount       = errors.New("el monto no es válido")
	ErrInsufficientFunds   = errors.New("el saldo disponible no alcanza")
	ErrReservationTooSmall = errors.New("la reserva no alcanza para liberar ese monto")
)

type Account struct {
	UserID           int64
	AvailableBalance Credits
	ReservedBalance  Credits
}

func (a *Account) Reserve(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.AvailableBalance < amount {
		return ErrInsufficientFunds
	}

	a.AvailableBalance -= amount
	a.ReservedBalance += amount

	return nil
}

func (a *Account) ReleaseReservation(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.ReservedBalance < amount {
		return ErrReservationTooSmall
	}

	a.ReservedBalance -= amount
	a.AvailableBalance += amount

	return nil
}

func (a *Account) SettleReservation(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.ReservedBalance < amount {
		return ErrReservationTooSmall
	}

	a.ReservedBalance -= amount

	return nil
}

func (a *Account) Credit(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	a.AvailableBalance += amount

	return nil
}
