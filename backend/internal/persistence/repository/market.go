package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

var ErrPlayerInactive = errors.New("player is not active")

const (
	ActionPlace   = "PLACE"
	ActionReserve = "RESERVE"
	ActionRelease = "RELEASE"
	ActionExecute = "EXECUTE"
	ActionSettle  = "SETTLE"
	ActionCancel  = "CANCEL"
)

type MarketOperation func(ctx context.Context, session *MarketSession) error

type MarketRepository struct {
	db          *sql.DB
	playerDao   *dao.PlayerSql
	accountDao  *dao.AccountSql
	positionDao *dao.PositionSql
	orderDao    *dao.OrderSql
	tradeDao    *dao.TradeSql
	ledgerDao   *dao.LedgerSql
	quoteDao    *dao.QuoteSql
	auditDao    dao.AuditDAO
	userDao     *dao.UserSql
}

func NewMarketRepository(
	db *sql.DB,
	playerDao *dao.PlayerSql,
	accountDao *dao.AccountSql,
	positionDao *dao.PositionSql,
	orderDao *dao.OrderSql,
	tradeDao *dao.TradeSql,
	ledgerDao *dao.LedgerSql,
	quoteDao *dao.QuoteSql,
	auditDao dao.AuditDAO,
	userDao *dao.UserSql,
) *MarketRepository {
	return &MarketRepository{
		db:          db,
		playerDao:   playerDao,
		accountDao:  accountDao,
		positionDao: positionDao,
		orderDao:    orderDao,
		tradeDao:    tradeDao,
		ledgerDao:   ledgerDao,
		quoteDao:    quoteDao,
		auditDao:    auditDao,
		userDao:     userDao,
	}
}

func (r *MarketRepository) WithPlayerLocked(ctx context.Context, playerID int64, operation MarketOperation) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir la transacción del mercado: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	active, lockErr := r.playerDao.LockPlayer(ctx, tx, playerID)
	if lockErr != nil {
		if errors.Is(lockErr, dao.ErrNotFound) {
			return ErrNotFound
		}
		return lockErr
	}
	if !active {
		return ErrPlayerInactive
	}

	session := &MarketSession{
		tx:          tx,
		accountDao:  r.accountDao,
		positionDao: r.positionDao,
		orderDao:    r.orderDao,
		tradeDao:    r.tradeDao,
		ledgerDao:   r.ledgerDao,
		quoteDao:    r.quoteDao,
		auditDao:    r.auditDao,
	}

	if operationErr := operation(ctx, session); operationErr != nil {
		return operationErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("confirmar la transacción del mercado: %w", commitErr)
	}

	return nil
}

func (r *MarketRepository) ActivePlayerIDs(ctx context.Context) ([]int64, error) {
	return r.playerDao.ListActiveIDs(ctx)
}

func (r *MarketRepository) MarketMaker(ctx context.Context) (model.User, error) {
	return r.userDao.GetSuperuser(ctx)
}

func (r *MarketRepository) Order(ctx context.Context, id int64) (model.Order, error) {
	return r.orderDao.GetByID(ctx, nil, id)
}

func (r *MarketRepository) LiveOrdersOf(ctx context.Context, userID int64, playerID *int64) ([]model.Order, error) {
	return r.orderDao.ListLiveByUser(ctx, userID, playerID)
}

func (r *MarketRepository) LiveMarketOrders(ctx context.Context, playerID int64) ([]model.Order, error) {
	return r.orderDao.ListLiveByPlayer(ctx, playerID)
}

type MarketSession struct {
	tx          *sql.Tx
	accountDao  *dao.AccountSql
	positionDao *dao.PositionSql
	orderDao    *dao.OrderSql
	tradeDao    *dao.TradeSql
	ledgerDao   *dao.LedgerSql
	quoteDao    *dao.QuoteSql
	auditDao    dao.AuditDAO
}

func (s *MarketSession) LockAccount(ctx context.Context, userID int64) (model.Account, error) {
	return s.accountDao.LockByUser(ctx, s.tx, userID)
}

func (s *MarketSession) LockPosition(ctx context.Context, userID, playerID int64) (model.Position, error) {
	return s.positionDao.LockByUserAndPlayer(ctx, s.tx, userID, playerID)
}

func (s *MarketSession) LockOppositeBook(ctx context.Context, playerID int64, incomingSide model.OrderSide) ([]model.Order, error) {
	return s.orderDao.LockLiveBook(ctx, s.tx, playerID, model.OppositeSide(incomingSide))
}

func (s *MarketSession) Order(ctx context.Context, id int64) (model.Order, error) {
	return s.orderDao.GetByID(ctx, s.tx, id)
}

func (s *MarketSession) InsertInitialQuote(ctx context.Context, playerID int64, price model.Credits) (bool, error) {
	return s.quoteDao.InsertInitialIfAbsent(ctx, s.tx, playerID, price)
}

func (s *MarketSession) RecordQuote(ctx context.Context, quote model.PlayerQuote) (model.PlayerQuote, error) {
	return s.quoteDao.Insert(ctx, s.tx, quote)
}

func (s *MarketSession) CreatePositionIfAbsent(ctx context.Context, position model.Position) (bool, error) {
	return s.positionDao.InsertIfAbsent(ctx, s.tx, position)
}

func (s *MarketSession) LiveMarketMakerOrder(ctx context.Context, playerID int64, side model.OrderSide) (model.Order, error) {
	return s.orderDao.GetLiveMarketMaker(ctx, s.tx, playerID, side)
}

func (s *MarketSession) LatestQuote(ctx context.Context, playerID int64) (model.PlayerQuote, error) {
	return s.quoteDao.GetLatestByPlayer(ctx, s.tx, playerID)
}

type OrderChange struct {
	Before model.Order
	After  model.Order
}

type AccountChange struct {
	Before model.Account
	After  model.Account
}

type PositionChange struct {
	Before model.Position
	After  model.Position
}

type ExecutionSettlement struct {
	Actor          string
	BuyOrder       OrderChange
	SellOrder      OrderChange
	BuyerPosition  PositionChange
	SellerPosition PositionChange
	BuyerAccount   AccountChange
	SellerAccount  AccountChange
	Quantity       int64
	Price          model.Credits
}

func (s *MarketSession) SettleExecution(ctx context.Context, settlement ExecutionSettlement) (model.Trade, error) {
	if err := s.applySettlementChanges(ctx, settlement); err != nil {
		return model.Trade{}, err
	}

	trade, err := s.tradeDao.Insert(ctx, s.tx, tradeOf(settlement))
	if err != nil {
		return model.Trade{}, err
	}

	if ledgerErr := s.recordTradeSettlement(ctx, trade); ledgerErr != nil {
		return model.Trade{}, ledgerErr
	}

	return trade, nil
}

func (s *MarketSession) applySettlementChanges(ctx context.Context, settlement ExecutionSettlement) error {
	if err := s.SaveOrder(ctx, settlement.Actor, ActionExecute, settlement.BuyOrder); err != nil {
		return err
	}
	if err := s.SaveOrder(ctx, settlement.Actor, ActionExecute, settlement.SellOrder); err != nil {
		return err
	}
	if err := s.SavePosition(ctx, settlement.Actor, ActionSettle, settlement.SellerPosition); err != nil {
		return err
	}
	if err := s.SavePosition(ctx, settlement.Actor, ActionSettle, settlement.BuyerPosition); err != nil {
		return err
	}
	if err := s.SaveAccount(ctx, settlement.Actor, ActionSettle, settlement.BuyerAccount); err != nil {
		return err
	}

	return s.SaveAccount(ctx, settlement.Actor, ActionSettle, settlement.SellerAccount)
}

func (s *MarketSession) InsertOrder(ctx context.Context, actor string, order model.Order) (model.Order, error) {
	stored, err := s.orderDao.Insert(ctx, s.tx, order)
	if err != nil {
		return model.Order{}, fmt.Errorf("registrar la orden del usuario %d: %w", order.UserID, err)
	}

	auditErr := s.auditDao.InsertAuditLog(ctx, s.tx, model.AuditLog{
		EntityType: "market_order",
		EntityID:   stored.ID,
		Action:     ActionPlace,
		Actor:      actor,
		DiffOld:    map[string]any{},
		DiffNew:    orderDiff(stored),
	})
	if auditErr != nil {
		return model.Order{}, auditErr
	}

	return stored, nil
}

func (s *MarketSession) SaveOrder(ctx context.Context, actor, action string, change OrderChange) error {
	if err := s.orderDao.Update(ctx, s.tx, change.After); err != nil {
		return fmt.Errorf("actualizar la orden %d: %w", change.After.ID, err)
	}

	return s.auditDao.InsertAuditLog(ctx, s.tx, model.AuditLog{
		EntityType: "market_order",
		EntityID:   change.After.ID,
		Action:     action,
		Actor:      actor,
		DiffOld:    orderDiff(change.Before),
		DiffNew:    orderDiff(change.After),
	})
}

func (s *MarketSession) SaveAccount(ctx context.Context, actor, action string, change AccountChange) error {
	if err := s.accountDao.UpdateBalances(ctx, s.tx, change.After); err != nil {
		return fmt.Errorf("actualizar la cuenta del usuario %d: %w", change.After.UserID, err)
	}

	return s.auditDao.InsertAuditLog(ctx, s.tx, model.AuditLog{
		EntityType: "user_account",
		EntityID:   change.After.UserID,
		Action:     action,
		Actor:      actor,
		DiffOld:    accountDiff(change.Before),
		DiffNew:    accountDiff(change.After),
	})
}

func (s *MarketSession) SavePosition(ctx context.Context, actor, action string, change PositionChange) error {
	if _, err := s.positionDao.InsertIfAbsent(ctx, s.tx, change.After); err != nil {
		return fmt.Errorf("crear la posición del usuario %d: %w", change.After.UserID, err)
	}
	if err := s.positionDao.Update(ctx, s.tx, change.After); err != nil {
		return fmt.Errorf("actualizar la posición del usuario %d: %w", change.After.UserID, err)
	}

	return s.auditDao.InsertAuditLog(ctx, s.tx, model.AuditLog{
		EntityType: "user_position",
		EntityID:   change.After.UserID,
		Action:     action,
		Actor:      actor,
		DiffOld:    positionDiff(change.Before),
		DiffNew:    positionDiff(change.After),
	})
}

func (s *MarketSession) recordTradeSettlement(ctx context.Context, trade model.Trade) error {
	settlementGroup, err := s.ledgerDao.NextSettlementGroup(ctx, s.tx)
	if err != nil {
		return err
	}

	tradeID := trade.ID
	buyerID := trade.BuyerID
	sellerID := trade.SellerID

	entries := []model.LedgerEntry{
		{
			TradeID:         &tradeID,
			UserID:          &buyerID,
			Reason:          model.ReasonTrade,
			Description:     fmt.Sprintf("compra de %d tokens del jugador %d", trade.Quantity, trade.PlayerID),
			SettlementGroup: settlementGroup,
			Amount:          -trade.TotalAmount,
		},
		{
			TradeID:         &tradeID,
			UserID:          &sellerID,
			Reason:          model.ReasonTrade,
			Description:     fmt.Sprintf("venta de %d tokens del jugador %d", trade.Quantity, trade.PlayerID),
			SettlementGroup: settlementGroup,
			Amount:          trade.TotalAmount,
		},
	}

	for i := range entries {
		if _, insertErr := s.ledgerDao.Insert(ctx, s.tx, entries[i]); insertErr != nil {
			return insertErr
		}
	}

	return nil
}

func tradeOf(settlement ExecutionSettlement) model.Trade {
	return model.Trade{
		BuyOrderID:  settlement.BuyOrder.After.ID,
		SellOrderID: settlement.SellOrder.After.ID,
		BuyerID:     settlement.BuyOrder.After.UserID,
		SellerID:    settlement.SellOrder.After.UserID,
		PlayerID:    settlement.BuyOrder.After.PlayerID,
		Quantity:    settlement.Quantity,
		Price:       settlement.Price,
		TotalAmount: settlement.Price.Times(settlement.Quantity),
	}
}

func orderDiff(order model.Order) map[string]any {
	return map[string]any{
		"status":           string(order.Status),
		"executedQuantity": order.ExecutedQuantity,
	}
}

func accountDiff(account model.Account) map[string]any {
	return map[string]any{
		"availableBalance": int64(account.AvailableBalance),
		"reservedBalance":  int64(account.ReservedBalance),
	}
}

func positionDiff(position model.Position) map[string]any {
	return map[string]any{
		"playerId":          position.PlayerID,
		"availableQuantity": position.AvailableQuantity,
		"reservedQuantity":  position.ReservedQuantity,
		"costBasis":         int64(position.CostBasis),
	}
}
