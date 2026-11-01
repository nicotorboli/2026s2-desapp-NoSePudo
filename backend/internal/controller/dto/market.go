package dto

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

var (
	ErrQuantityNotPositive = errors.New("quantity debe ser un entero mayor a cero")
	ErrPriceNotPositive    = errors.New("price debe ser mayor a cero")
	ErrPriceNotANumber     = errors.New("price no es un número decimal válido")
	ErrPriceTooPrecise     = errors.New("price admite a lo sumo cuatro decimales")
	ErrUnknownTimeInForce  = errors.New("timeInForce debe ser GTC, IOC o FOK")
	ErrPlayerNotIdentified = errors.New("playerId es obligatorio")
)

func CreditsADecimal(credits model.Credits) string {
	negative := credits < 0
	units := int64(credits)
	if negative {
		units = -units
	}

	whole := units / model.CreditScale
	fraction := units % model.CreditScale

	rendered := fmt.Sprintf("%d.%04d", whole, fraction)
	if negative {
		rendered = "-" + rendered
	}

	return rendered
}

func CreditsDesdeDecimal(raw string) (model.Credits, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, ErrPriceNotANumber
	}

	negative := strings.HasPrefix(trimmed, "-")
	trimmed = strings.TrimPrefix(trimmed, "-")

	whole, fraction, hasFraction := strings.Cut(trimmed, ".")
	if hasFraction && len(fraction) > 4 {
		return 0, ErrPriceTooPrecise
	}

	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrPriceNotANumber
	}

	fractionUnits, err := fractionToUnits(fraction)
	if err != nil {
		return 0, err
	}

	if wholeUnits > math.MaxInt64/model.CreditScale {
		return 0, ErrPriceNotANumber
	}

	credits := model.Credits(wholeUnits*model.CreditScale + fractionUnits)
	if negative {
		credits = -credits
	}

	return credits, nil
}

func fractionToUnits(fraction string) (int64, error) {
	if fraction == "" {
		return 0, nil
	}

	padded := fraction + strings.Repeat("0", 4-len(fraction))

	units, err := strconv.ParseInt(padded, 10, 64)
	if err != nil {
		return 0, ErrPriceNotANumber
	}

	return units, nil
}

type PlaceOrderRequest struct {
	Price       *string `json:"price"`
	TimeInForce *string `json:"timeInForce"`
	PlayerID    int64   `json:"playerId"`
	Quantity    int64   `json:"quantity"`
}

func (r *PlaceOrderRequest) Validate() error {
	if r.PlayerID <= 0 {
		return ErrPlayerNotIdentified
	}
	if r.Quantity <= 0 {
		return ErrQuantityNotPositive
	}
	if _, err := r.ParsedPrice(); err != nil {
		return err
	}

	return r.validateTimeInForce()
}

func (r *PlaceOrderRequest) ParsedPrice() (*model.Credits, error) {
	if r.Price == nil {
		return nil, nil
	}

	price, err := CreditsDesdeDecimal(*r.Price)
	if err != nil {
		return nil, err
	}
	if price <= 0 {
		return nil, ErrPriceNotPositive
	}

	return &price, nil
}

func (r *PlaceOrderRequest) validateTimeInForce() error {
	if r.TimeInForce == nil {
		return nil
	}

	switch model.TimeInForce(*r.TimeInForce) {
	case model.GoodTillCancelled, model.ImmediateOrCancel, model.FillOrKill:
		return nil
	}

	return ErrUnknownTimeInForce
}

func (r *PlaceOrderRequest) ParsedTimeInForce() *model.TimeInForce {
	if r.TimeInForce == nil {
		return nil
	}

	parsed := model.TimeInForce(*r.TimeInForce)

	return &parsed
}

type TradeResponse struct {
	ExecutedAt  time.Time `json:"executedAt"`
	Price       string    `json:"price"`
	TotalAmount string    `json:"totalAmount"`
	Quantity    int64     `json:"quantity"`
}

func TradeResponseDesdeModelo(trade model.Trade) TradeResponse {
	return TradeResponse{
		ExecutedAt:  trade.ExecutedAt,
		Price:       CreditsADecimal(trade.Price),
		TotalAmount: CreditsADecimal(trade.TotalAmount),
		Quantity:    trade.Quantity,
	}
}

type OrderResponse struct {
	CreatedAt         time.Time       `json:"createdAt"`
	Side              model.OrderSide `json:"side"`
	TimeInForce       string          `json:"timeInForce"`
	Status            string          `json:"status"`
	Price             string          `json:"price"`
	Trades            []TradeResponse `json:"trades,omitempty"`
	ID                int64           `json:"id"`
	PlayerID          int64           `json:"playerId"`
	UserID            int64           `json:"userId,omitempty"`
	Quantity          int64           `json:"quantity"`
	ExecutedQuantity  int64           `json:"executedQuantity"`
	RemainingQuantity int64           `json:"remainingQuantity"`
}

func OrderResponseDesdeModelo(order model.Order, trades []model.Trade) OrderResponse {
	response := OrderResponse{
		CreatedAt:         order.CreatedAt,
		Side:              order.Side,
		TimeInForce:       string(order.TimeInForce),
		Status:            string(order.Status),
		Price:             CreditsADecimal(order.Price),
		ID:                order.ID,
		PlayerID:          order.PlayerID,
		UserID:            order.UserID,
		Quantity:          order.Quantity,
		ExecutedQuantity:  order.ExecutedQuantity,
		RemainingQuantity: order.RemainingQuantity(),
	}

	if len(trades) > 0 {
		response.Trades = make([]TradeResponse, 0, len(trades))
		for i := range trades {
			response.Trades = append(response.Trades, TradeResponseDesdeModelo(trades[i]))
		}
	}

	return response
}

type OrderListResponse struct {
	Items []OrderResponse `json:"items"`
}

func OrderListResponseDesdeModelo(orders []model.Order) OrderListResponse {
	items := make([]OrderResponse, 0, len(orders))
	for i := range orders {
		items = append(items, OrderResponseDesdeModelo(orders[i], nil))
	}

	return OrderListResponse{Items: items}
}

type CancelOrderResponse struct {
	ReleasedCredits *string `json:"releasedCredits,omitempty"`
	ReleasedTokens  *int64  `json:"releasedTokens,omitempty"`
	Status          string  `json:"status"`
	ID              int64   `json:"id"`
}

func CancelOrderResponseDesdeModelo(cancelled service.CancelledOrder) CancelOrderResponse {
	response := CancelOrderResponse{
		Status: string(cancelled.Order.Status),
		ID:     cancelled.Order.ID,
	}

	if cancelled.ReleasedCredits > 0 {
		rendered := CreditsADecimal(cancelled.ReleasedCredits)
		response.ReleasedCredits = &rendered
	}
	if cancelled.ReleasedTokens > 0 {
		released := cancelled.ReleasedTokens
		response.ReleasedTokens = &released
	}

	return response
}

type PortfolioPositionResponse struct {
	PlayerName        string `json:"playerName"`
	ClubName          string `json:"clubName"`
	Position          string `json:"position"`
	AveragePrice      string `json:"averagePrice"`
	CurrentPrice      string `json:"currentPrice"`
	CurrentValuation  string `json:"currentValuation"`
	UnrealizedResult  string `json:"unrealizedResult"`
	PlayerID          int64  `json:"playerId"`
	Quantity          int64  `json:"quantity"`
	AvailableQuantity int64  `json:"availableQuantity"`
	ReservedQuantity  int64  `json:"reservedQuantity"`
}

type PortfolioResponse struct {
	AvailableCash    string                      `json:"availableCash"`
	ReservedCash     string                      `json:"reservedCash"`
	TokensValuation  string                      `json:"tokensValuation"`
	TotalValue       string                      `json:"totalValue"`
	UnrealizedResult string                      `json:"unrealizedResult"`
	Positions        []PortfolioPositionResponse `json:"positions"`
	UserID           int64                       `json:"userId"`
}

func PortfolioResponseDesdeModelo(portfolio model.Portfolio) PortfolioResponse {
	positions := make([]PortfolioPositionResponse, 0, len(portfolio.Positions))
	for i := range portfolio.Positions {
		positions = append(positions, portfolioPositionDesdeModelo(portfolio.Positions[i]))
	}

	return PortfolioResponse{
		AvailableCash:    CreditsADecimal(portfolio.AvailableCash),
		ReservedCash:     CreditsADecimal(portfolio.ReservedCash),
		TokensValuation:  CreditsADecimal(portfolio.TokensValuation),
		TotalValue:       CreditsADecimal(portfolio.TotalValue),
		UnrealizedResult: CreditsADecimal(portfolio.UnrealizedResult),
		Positions:        positions,
		UserID:           portfolio.UserID,
	}
}

func portfolioPositionDesdeModelo(position model.PortfolioPosition) PortfolioPositionResponse {
	return PortfolioPositionResponse{
		PlayerName:        position.PlayerName,
		ClubName:          position.ClubName,
		Position:          position.PlayerPosition,
		AveragePrice:      CreditsADecimal(position.AveragePrice),
		CurrentPrice:      CreditsADecimal(position.CurrentPrice),
		CurrentValuation:  CreditsADecimal(position.CurrentValuation),
		UnrealizedResult:  CreditsADecimal(position.UnrealizedResult),
		PlayerID:          position.PlayerID,
		Quantity:          position.Quantity,
		AvailableQuantity: position.AvailableQuantity,
		ReservedQuantity:  position.ReservedQuantity,
	}
}

type TransactionResponse struct {
	ExecutedAt  time.Time       `json:"executedAt"`
	PlayerName  string          `json:"playerName"`
	Side        model.OrderSide `json:"side"`
	Price       string          `json:"price"`
	TotalAmount string          `json:"totalAmount"`
	ID          int64           `json:"id"`
	PlayerID    int64           `json:"playerId"`
	Quantity    int64           `json:"quantity"`
}

func TransactionResponseDesdeModelo(entry service.TransactionEntry) TransactionResponse {
	return TransactionResponse{
		ExecutedAt:  entry.ExecutedAt,
		PlayerName:  entry.PlayerName,
		Side:        entry.Side,
		Price:       CreditsADecimal(entry.Price),
		TotalAmount: CreditsADecimal(entry.TotalAmount),
		ID:          entry.ID,
		PlayerID:    entry.PlayerID,
		Quantity:    entry.Quantity,
	}
}

type TransactionPageResponse struct {
	Items      []TransactionResponse `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"totalPages"`
}

func TransactionPageResponseDesdeModelo(page model.PageResult[service.TransactionEntry]) TransactionPageResponse {
	items := make([]TransactionResponse, 0, len(page.Items))
	for i := range page.Items {
		items = append(items, TransactionResponseDesdeModelo(page.Items[i]))
	}

	return TransactionPageResponse{
		Items:      items,
		Total:      page.Total,
		Page:       page.Page,
		Limit:      page.Limit,
		TotalPages: page.TotalPages,
	}
}
