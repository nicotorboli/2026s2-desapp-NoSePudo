package dto

import (
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var (
	ErrP2PPlayerRequired   = errors.New("playerId es obligatorio y debe ser mayor a cero")
	ErrP2PQuantityRequired = errors.New("quantity debe ser mayor a cero")
	ErrP2PPriceRequired    = errors.New("pricePerToken debe ser mayor a cero")
)

type CreateP2PSessionRequest struct {
	PlayerID      int64 `json:"playerId"`
	Quantity      int64 `json:"quantity"`
	PricePerToken int64 `json:"pricePerToken"`
}

func (r *CreateP2PSessionRequest) Validate() error {
	if r.PlayerID <= 0 {
		return ErrP2PPlayerRequired
	}
	if r.Quantity <= 0 {
		return ErrP2PQuantityRequired
	}
	if r.PricePerToken <= 0 {
		return ErrP2PPriceRequired
	}
	return nil
}

type P2PUserSummary struct {
	Email string `json:"email"`
	ID    int64  `json:"id"`
}

type P2PPlayerSummary struct {
	Name     string `json:"name"`
	ClubName string `json:"clubName"`
	ID       int64  `json:"id"`
}

type P2PSessionResponse struct {
	ExpiresAt     time.Time        `json:"expiresAt"`
	CreatedAt     time.Time        `json:"createdAt"`
	CompletedAt   *time.Time       `json:"completedAt,omitempty"`
	Token         string           `json:"token"`
	SessionURL    string           `json:"sessionUrl"`
	Status        string           `json:"status"`
	Seller        P2PUserSummary   `json:"seller"`
	Buyer         *P2PUserSummary  `json:"buyer"`
	Player        P2PPlayerSummary `json:"player"`
	ID            int64            `json:"id"`
	Quantity      int64            `json:"quantity"`
	PricePerToken int64            `json:"pricePerToken"`
	TotalAmount   int64            `json:"totalAmount"`
}

func P2PSessionResponseDesdeModelo(session model.P2PSession, player model.Player, sessionURL string) P2PSessionResponse {
	if sessionURL == "" {
		sessionURL = fmt.Sprintf("/?token=%s", session.Token)
	}

	var buyer *P2PUserSummary
	if session.BuyerID != nil {
		buyer = &P2PUserSummary{
			ID:    *session.BuyerID,
			Email: session.BuyerEmail,
		}
	}

	return P2PSessionResponse{
		ID:         session.ID,
		Token:      session.Token,
		SessionURL: sessionURL,
		Seller: P2PUserSummary{
			ID:    session.SellerID,
			Email: session.SellerEmail,
		},
		Buyer: buyer,
		Player: P2PPlayerSummary{
			ID:       player.ID,
			Name:     player.Name,
			ClubName: player.ClubName,
		},
		Quantity:      session.Quantity,
		PricePerToken: int64(session.Price),
		TotalAmount:   int64(session.TotalAmount()),
		Status:        string(session.Status),
		CreatedAt:     session.CreatedAt,
		ExpiresAt:     session.ExpiresAt,
		CompletedAt:   session.CompletedAt,
	}
}

type P2PTradeResponse struct {
	ExecutedAt    time.Time `json:"executedAt"`
	SessionToken  string    `json:"sessionToken"`
	TradeID       int64     `json:"tradeId"`
	BuyerID       int64     `json:"buyerId"`
	SellerID      int64     `json:"sellerId"`
	PlayerID      int64     `json:"playerId"`
	Quantity      int64     `json:"quantity"`
	PricePerToken int64     `json:"pricePerToken"`
	TotalAmount   int64     `json:"totalAmount"`
}

func P2PTradeResponseDesdeModelo(trade model.Trade, sessionToken string) P2PTradeResponse {
	return P2PTradeResponse{
		TradeID:       trade.ID,
		SessionToken:  sessionToken,
		BuyerID:       trade.BuyerID,
		SellerID:      trade.SellerID,
		PlayerID:      trade.PlayerID,
		Quantity:      trade.Quantity,
		PricePerToken: int64(trade.Price),
		TotalAmount:   int64(trade.TotalAmount),
		ExecutedAt:    trade.ExecutedAt,
	}
}
