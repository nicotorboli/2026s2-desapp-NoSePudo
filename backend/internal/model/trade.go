package model

import "time"

type Trade struct {
	ExecutedAt   time.Time
	P2PSessionID *int64
	ID           int64
	BuyOrderID   int64
	SellOrderID  int64
	BuyerID      int64
	SellerID     int64
	PlayerID     int64
	Quantity     int64
	Price        Credits
	TotalAmount  Credits
}

type LedgerReason string

const (
	ReasonTrade        LedgerReason = "TRADE"
	ReasonInitialGrant LedgerReason = "INITIAL_GRANT"
)

type LedgerEntry struct {
	CreatedAt       time.Time
	TradeID         *int64
	UserID          *int64
	Reason          LedgerReason
	Description     string
	ID              int64
	SettlementGroup int64
	Amount          Credits
}
