package model

import (
	"errors"
	"time"
)

var (
	ErrOrderNotLive    = errors.New("la orden ya no está viva")
	ErrInvalidQuantity = errors.New("la cantidad no es válida")
)

type OrderSide string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"
)

type TimeInForce string

const (
	GoodTillCancelled TimeInForce = "GTC"
	ImmediateOrCancel TimeInForce = "IOC"
	FillOrKill        TimeInForce = "FOK"
)

type OrderStatus string

const (
	StatusPending         OrderStatus = "PENDING"
	StatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	StatusFilled          OrderStatus = "FILLED"
	StatusCancelled       OrderStatus = "CANCELLED"
)

type Order struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Side             OrderSide
	TimeInForce      TimeInForce
	Status           OrderStatus
	ID               int64
	UserID           int64
	PlayerID         int64
	Quantity         int64
	ExecutedQuantity int64
	Price            Credits
	MarketMaker      bool
}

func (o *Order) RemainingQuantity() int64 {
	return o.Quantity - o.ExecutedQuantity
}

func (o *Order) IsLive() bool {
	return o.Status == StatusPending || o.Status == StatusPartiallyFilled
}

func (o *Order) Execute(quantity int64, at time.Time) error {
	if !o.IsLive() {
		return ErrOrderNotLive
	}
	if quantity <= 0 || quantity > o.RemainingQuantity() {
		return ErrInvalidQuantity
	}

	o.ExecutedQuantity += quantity
	if o.RemainingQuantity() == 0 {
		o.Status = StatusFilled
	} else {
		o.Status = StatusPartiallyFilled
	}
	o.UpdatedAt = at

	return nil
}

func (o *Order) Cancel(at time.Time) error {
	if !o.IsLive() {
		return ErrOrderNotLive
	}

	o.Status = StatusCancelled
	o.UpdatedAt = at

	return nil
}

func OppositeSide(side OrderSide) OrderSide {
	if side == SideBuy {
		return SideSell
	}
	return SideBuy
}
