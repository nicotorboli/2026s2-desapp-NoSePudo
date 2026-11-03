package model

import (
	"errors"
	"time"
)

var (
	ErrP2PSessionNotFound     = errors.New("no existe una sesión P2P con ese token")
	ErrP2PSessionNotLive      = errors.New("la sesión P2P ya no está activa")
	ErrP2PSessionExpired      = errors.New("la oferta venció y ya no puede aceptarse")
	ErrP2PSessionAlreadyTaken = errors.New("la oferta ya fue tomada por otro usuario")
	ErrP2PSessionSelfTrade    = errors.New("el vendedor no puede aceptar su propia oferta")
)

type P2PSessionStatus string

const (
	P2PStatusPending   P2PSessionStatus = "PENDING"
	P2PStatusCompleted P2PSessionStatus = "COMPLETED"
	P2PStatusCancelled P2PSessionStatus = "CANCELLED"
	P2PStatusExpired   P2PSessionStatus = "EXPIRED"
)

type P2PSession struct {
	ExpiresAt   time.Time
	CreatedAt   time.Time
	CompletedAt *time.Time
	BuyerID     *int64
	Status      P2PSessionStatus
	BuyerEmail  string
	SellerEmail string
	Token       string
	ID          int64
	SellerID    int64
	PlayerID    int64
	Quantity    int64
	Price       Credits
}

func (s *P2PSession) IsLive() bool {
	return s.Status == P2PStatusPending
}

func (s *P2PSession) IsExpired(now time.Time) bool {
	return s.Status == P2PStatusPending && (now.Equal(s.ExpiresAt) || now.After(s.ExpiresAt))
}

func (s *P2PSession) TotalAmount() Credits {
	return s.Price.Times(s.Quantity)
}

func (s *P2PSession) Accept(buyerID int64, now time.Time) error {
	if s.Status == P2PStatusExpired || (s.Status == P2PStatusPending && s.IsExpired(now)) {
		return ErrP2PSessionExpired
	}
	if s.Status == P2PStatusCompleted {
		return ErrP2PSessionAlreadyTaken
	}
	if !s.IsLive() {
		return ErrP2PSessionNotLive
	}
	if s.SellerID == buyerID {
		return ErrP2PSessionSelfTrade
	}

	s.BuyerID = &buyerID
	s.Status = P2PStatusCompleted
	t := now
	s.CompletedAt = &t
	return nil
}

func (s *P2PSession) Cancel(now time.Time) error {
	if !s.IsLive() {
		return ErrP2PSessionNotLive
	}

	s.Status = P2PStatusCancelled
	t := now
	s.CompletedAt = &t
	return nil
}

func (s *P2PSession) Expire(now time.Time) error {
	if !s.IsLive() {
		return ErrP2PSessionNotLive
	}

	s.Status = P2PStatusExpired
	t := now
	s.CompletedAt = &t
	return nil
}
