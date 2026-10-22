package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var (
	ErrInvalidQuantity = errors.New("la cantidad debe ser mayor a cero")
	ErrInvalidPrice    = errors.New("el precio debe ser mayor a cero")
	ErrPlayerInactive  = errors.New("el jugador no está activo")
	ErrNoLiquidity     = errors.New("la orden no se puede completar en su totalidad")

	ErrSellPathUnavailable = errors.New("la venta todavía no está habilitada")
)

type PlaceOrderCommand struct {
	Price       *model.Credits
	TimeInForce *model.TimeInForce
	Actor       string
	Side        model.OrderSide
	UserID      int64
	PlayerID    int64
	Quantity    int64
}

type PlacedOrder struct {
	Trades []model.Trade
	Order  model.Order
}

type MarketService struct {
	market *repository.MarketRepository
	now    func() time.Time
}

func NewMarketService(market *repository.MarketRepository, now func() time.Time) *MarketService {
	return &MarketService{market: market, now: now}
}

func (s *MarketService) PlaceOrder(ctx context.Context, command PlaceOrderCommand) (PlacedOrder, error) {
	if err := validatePlaceOrder(command); err != nil {
		return PlacedOrder{}, err
	}

	var placed PlacedOrder

	err := s.market.WithPlayerLocked(ctx, command.PlayerID, func(ctx context.Context, session *repository.MarketSession) error {
		result, placeErr := s.placeWithinLock(ctx, session, command)
		if placeErr != nil {
			return placeErr
		}
		placed = result

		return nil
	})
	if err != nil {
		return PlacedOrder{}, translateMarketError(err)
	}

	return placed, nil
}

func validatePlaceOrder(command PlaceOrderCommand) error {
	if command.Side != model.SideBuy {
		return ErrSellPathUnavailable
	}
	if command.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if command.Price != nil && *command.Price <= 0 {
		return ErrInvalidPrice
	}

	return nil
}

func (s *MarketService) placeWithinLock(ctx context.Context, session *repository.MarketSession, command PlaceOrderCommand) (PlacedOrder, error) {
	price, err := resolvePrice(ctx, session, command)
	if err != nil {
		return PlacedOrder{}, err
	}

	account, err := s.reserveCredits(ctx, session, command, price)
	if err != nil {
		return PlacedOrder{}, err
	}

	incoming, err := session.InsertOrder(ctx, command.Actor, model.Order{
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
		Side:        command.Side,
		TimeInForce: resolveTimeInForce(command),
		Status:      model.StatusPending,
		UserID:      command.UserID,
		PlayerID:    command.PlayerID,
		Quantity:    command.Quantity,
		Price:       price,
	})
	if err != nil {
		return PlacedOrder{}, err
	}

	book, err := session.LockOppositeBook(ctx, command.PlayerID, command.Side)
	if err != nil {
		return PlacedOrder{}, err
	}

	trades, err := s.settleExecutions(ctx, session, command, &incoming, &account, book)
	if err != nil {
		return PlacedOrder{}, err
	}

	if err := s.applyRemainder(ctx, session, command, &incoming, &account); err != nil {
		return PlacedOrder{}, err
	}

	return PlacedOrder{Order: incoming, Trades: trades}, nil
}

func resolvePrice(ctx context.Context, session *repository.MarketSession, command PlaceOrderCommand) (model.Credits, error) {
	if command.Price != nil {
		return *command.Price, nil
	}

	quote, err := session.LatestQuote(ctx, command.PlayerID)
	if err != nil {
		return 0, err
	}

	return quote.Price, nil
}

func resolveTimeInForce(command PlaceOrderCommand) model.TimeInForce {
	if command.TimeInForce != nil {
		return *command.TimeInForce
	}
	if command.Price == nil {
		return model.FillOrKill
	}

	return model.GoodTillCancelled
}

func (s *MarketService) reserveCredits(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	price model.Credits,
) (model.Account, error) {
	account, err := session.LockAccount(ctx, command.UserID)
	if err != nil {
		return model.Account{}, err
	}

	before := account
	if reserveErr := account.Reserve(price.Times(command.Quantity)); reserveErr != nil {
		return model.Account{}, reserveErr
	}

	change := repository.AccountChange{Before: before, After: account}
	if saveErr := session.SaveAccount(ctx, command.Actor, repository.ActionReserve, change); saveErr != nil {
		return model.Account{}, saveErr
	}

	return account, nil
}

func (s *MarketService) settleExecutions(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	account *model.Account,
	book []model.Order,
) ([]model.Trade, error) {
	executions := model.MatchOrder(*incoming, book)
	trades := make([]model.Trade, 0, len(executions))

	for _, execution := range executions {
		resting, found := findOrder(book, execution.RestingOrderID)
		if !found {
			return nil, fmt.Errorf("la orden en reposo %d no está en el libro leído", execution.RestingOrderID)
		}

		trade, err := s.settleOne(ctx, session, command, incoming, account, resting, execution)
		if err != nil {
			return nil, err
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func findOrder(book []model.Order, id int64) (model.Order, bool) {
	for i := range book {
		if book[i].ID == id {
			return book[i], true
		}
	}

	return model.Order{}, false
}

func (s *MarketService) settleOne(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	account *model.Account,
	resting model.Order,
	execution model.Execution,
) (model.Trade, error) {
	at := s.now()

	incomingBefore := *incoming
	if err := incoming.Execute(execution.Quantity, at); err != nil {
		return model.Trade{}, err
	}

	restingBefore := resting
	if err := resting.Execute(execution.Quantity, at); err != nil {
		return model.Trade{}, err
	}

	buyer, seller := settlementParties{
		order:   OrderChangeOf(incomingBefore, *incoming),
		userID:  command.UserID,
		account: account,
	}, settlementParties{
		order:  OrderChangeOf(restingBefore, resting),
		userID: resting.UserID,
	}

	return s.exchange(ctx, session, command, buyer, seller, execution)
}

type settlementParties struct {
	account *model.Account
	order   repository.OrderChange
	userID  int64
}

func OrderChangeOf(before, after model.Order) repository.OrderChange {
	return repository.OrderChange{Before: before, After: after}
}

func (s *MarketService) exchange(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	buyer settlementParties,
	seller settlementParties,
	execution model.Execution,
) (model.Trade, error) {
	amount := execution.Price.Times(execution.Quantity)

	buyerAccountBefore := *buyer.account
	if err := buyer.account.SettleReservation(amount); err != nil {
		return model.Trade{}, err
	}
	if improvement := buyer.order.After.Price.Minus(execution.Price); improvement > 0 {
		if err := buyer.account.ReleaseReservation(improvement.Times(execution.Quantity)); err != nil {
			return model.Trade{}, err
		}
	}

	sellerAccount, err := session.LockAccount(ctx, seller.userID)
	if err != nil {
		return model.Trade{}, err
	}
	sellerAccountBefore := sellerAccount
	if creditErr := sellerAccount.Credit(amount); creditErr != nil {
		return model.Trade{}, creditErr
	}

	sellerPosition, err := session.LockPosition(ctx, seller.userID, command.PlayerID)
	if err != nil {
		return model.Trade{}, err
	}
	sellerPositionBefore := sellerPosition
	if _, deliverErr := sellerPosition.DeliverReservedTokens(execution.Quantity); deliverErr != nil {
		return model.Trade{}, deliverErr
	}

	buyerPosition, err := loadOrEmptyPosition(ctx, session, buyer.userID, command.PlayerID)
	if err != nil {
		return model.Trade{}, err
	}
	buyerPositionBefore := buyerPosition
	if receiveErr := buyerPosition.Receive(execution.Quantity, amount); receiveErr != nil {
		return model.Trade{}, receiveErr
	}

	return session.SettleExecution(ctx, repository.ExecutionSettlement{
		Actor:          command.Actor,
		BuyOrder:       buyer.order,
		SellOrder:      seller.order,
		BuyerPosition:  repository.PositionChange{Before: buyerPositionBefore, After: buyerPosition},
		SellerPosition: repository.PositionChange{Before: sellerPositionBefore, After: sellerPosition},
		BuyerAccount:   repository.AccountChange{Before: buyerAccountBefore, After: *buyer.account},
		SellerAccount:  repository.AccountChange{Before: sellerAccountBefore, After: sellerAccount},
		Quantity:       execution.Quantity,
		Price:          execution.Price,
	})
}

func loadOrEmptyPosition(
	ctx context.Context,
	session *repository.MarketSession,
	userID, playerID int64,
) (model.Position, error) {
	position, err := session.LockPosition(ctx, userID, playerID)
	if err == nil {
		return position, nil
	}
	if errors.Is(err, model.ErrPositionNotFound) {
		return model.Position{UserID: userID, PlayerID: playerID}, nil
	}

	return model.Position{}, err
}

func (s *MarketService) applyRemainder(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	account *model.Account,
) error {
	remaining := incoming.RemainingQuantity()
	if remaining == 0 {
		return nil
	}

	switch incoming.TimeInForce {
	case model.GoodTillCancelled:
		return nil
	case model.FillOrKill:
		return ErrNoLiquidity
	case model.ImmediateOrCancel:
		return s.cancelRemainder(ctx, session, command, incoming, account, remaining)
	}

	return fmt.Errorf("vigencia desconocida: %s", incoming.TimeInForce)
}

func (s *MarketService) cancelRemainder(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	account *model.Account,
	remaining int64,
) error {
	before := *account
	if err := account.ReleaseReservation(incoming.Price.Times(remaining)); err != nil {
		return err
	}

	releaseChange := repository.AccountChange{Before: before, After: *account}
	if err := session.SaveAccount(ctx, command.Actor, repository.ActionRelease, releaseChange); err != nil {
		return err
	}

	orderBefore := *incoming
	if err := incoming.Cancel(s.now()); err != nil {
		return err
	}

	orderChange := repository.OrderChange{Before: orderBefore, After: *incoming}

	return session.SaveOrder(ctx, command.Actor, repository.ActionCancel, orderChange)
}

func translateMarketError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrPlayerInactive):
		return ErrPlayerInactive
	default:
		return err
	}
}
