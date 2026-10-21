package model

import "errors"

var (
	ErrInsufficientTokens = errors.New("la posición disponible no alcanza")
	ErrPositionNotFound   = errors.New("no existe una posición de ese usuario en ese jugador")
)

type Position struct {
	UserID            int64
	PlayerID          int64
	AvailableQuantity int64
	ReservedQuantity  int64
	CostBasis         Credits
}

func (p *Position) TotalQuantity() int64 {
	return p.AvailableQuantity + p.ReservedQuantity
}

func (p *Position) AveragePrice() Credits {
	return p.CostBasis.DividedBy(p.TotalQuantity())
}

func (p *Position) ReserveTokens(quantity int64) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if p.AvailableQuantity < quantity {
		return ErrInsufficientTokens
	}

	p.AvailableQuantity -= quantity
	p.ReservedQuantity += quantity

	return nil
}

func (p *Position) ReleaseReservedTokens(quantity int64) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if p.ReservedQuantity < quantity {
		return ErrReservationTooSmall
	}

	p.ReservedQuantity -= quantity
	p.AvailableQuantity += quantity

	return nil
}

func (p *Position) DeliverReservedTokens(quantity int64) (Credits, error) {
	if quantity <= 0 {
		return 0, ErrInvalidQuantity
	}
	if p.ReservedQuantity < quantity {
		return 0, ErrReservationTooSmall
	}

	released := p.CostBasis
	if quantity < p.TotalQuantity() {
		released = p.CostBasis.Times(quantity).DividedBy(p.TotalQuantity())
	}

	p.ReservedQuantity -= quantity
	p.CostBasis -= released

	return released, nil
}

func (p *Position) Receive(quantity int64, cost Credits) error {
	if quantity <= 0 || cost <= 0 {
		return ErrInvalidQuantity
	}

	p.AvailableQuantity += quantity
	p.CostBasis += cost

	return nil
}
