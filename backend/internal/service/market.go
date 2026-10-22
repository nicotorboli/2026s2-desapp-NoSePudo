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
	ErrUnknownSide     = errors.New("el lado de la orden es desconocido")
	ErrNotOwner        = errors.New("la orden pertenece a otro usuario")
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
	if command.Side != model.SideBuy && command.Side != model.SideSell {
		return ErrUnknownSide
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

	submitter, err := loadParty(ctx, session, command.UserID, command.PlayerID)
	if err != nil {
		return PlacedOrder{}, err
	}

	if reserveErr := s.reserve(ctx, session, command, submitter, price); reserveErr != nil {
		return PlacedOrder{}, reserveErr
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

	trades, err := s.settleExecutions(ctx, session, command, &incoming, submitter, book)
	if err != nil {
		return PlacedOrder{}, err
	}

	if err := s.applyRemainder(ctx, session, command, &incoming, submitter); err != nil {
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

type tradingParty struct {
	account  model.Account
	position model.Position
	userID   int64
}

func loadParty(ctx context.Context, session *repository.MarketSession, userID, playerID int64) (*tradingParty, error) {
	account, err := session.LockAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	position, err := session.LockPosition(ctx, userID, playerID)
	if err != nil {
		if !errors.Is(err, model.ErrPositionNotFound) {
			return nil, err
		}
		position = model.Position{UserID: userID, PlayerID: playerID}
	}

	return &tradingParty{userID: userID, account: account, position: position}, nil
}

func (s *MarketService) reserve(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	submitter *tradingParty,
	price model.Credits,
) error {
	if command.Side == model.SideBuy {
		before := submitter.account
		if err := submitter.account.Reserve(price.Times(command.Quantity)); err != nil {
			return err
		}

		change := repository.AccountChange{Before: before, After: submitter.account}

		return session.SaveAccount(ctx, command.Actor, repository.ActionReserve, change)
	}

	before := submitter.position
	if err := submitter.position.ReserveTokens(command.Quantity); err != nil {
		return err
	}

	change := repository.PositionChange{Before: before, After: submitter.position}

	return session.SavePosition(ctx, command.Actor, repository.ActionReserve, change)
}

func (s *MarketService) settleExecutions(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	submitter *tradingParty,
	book []model.Order,
) ([]model.Trade, error) {
	executions := model.MatchOrder(*incoming, book)
	trades := make([]model.Trade, 0, len(executions))
	counterparties := make(map[int64]*tradingParty, len(executions))

	for _, execution := range executions {
		resting, found := findOrder(book, execution.RestingOrderID)
		if !found {
			return nil, fmt.Errorf("la orden en reposo %d no está en el libro leído", execution.RestingOrderID)
		}

		counterparty, err := partyOf(ctx, session, counterparties, resting.UserID, command.PlayerID)
		if err != nil {
			return nil, err
		}

		trade, err := s.settleOne(ctx, session, command, incoming, submitter, resting, counterparty, execution)
		if err != nil {
			return nil, err
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func partyOf(
	ctx context.Context,
	session *repository.MarketSession,
	known map[int64]*tradingParty,
	userID, playerID int64,
) (*tradingParty, error) {
	if party, seen := known[userID]; seen {
		return party, nil
	}

	party, err := loadParty(ctx, session, userID, playerID)
	if err != nil {
		return nil, err
	}
	known[userID] = party

	return party, nil
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
	submitter *tradingParty,
	resting model.Order,
	counterparty *tradingParty,
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

	incomingChange := repository.OrderChange{Before: incomingBefore, After: *incoming}
	restingChange := repository.OrderChange{Before: restingBefore, After: resting}

	sides := settlementSides{
		buyOrder:    incomingChange,
		sellOrder:   restingChange,
		buyerParty:  submitter,
		sellerParty: counterparty,
	}
	if command.Side == model.SideSell {
		sides = settlementSides{
			buyOrder:    restingChange,
			sellOrder:   incomingChange,
			buyerParty:  counterparty,
			sellerParty: submitter,
		}
	}

	return s.exchange(ctx, session, command, sides, execution)
}

type settlementSides struct {
	buyerParty  *tradingParty
	sellerParty *tradingParty
	buyOrder    repository.OrderChange
	sellOrder   repository.OrderChange
}

func (s *MarketService) exchange(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	sides settlementSides,
	execution model.Execution,
) (model.Trade, error) {
	amount := execution.Price.Times(execution.Quantity)

	buyerAccountBefore := sides.buyerParty.account
	if err := sides.buyerParty.account.SettleReservation(amount); err != nil {
		return model.Trade{}, err
	}
	if improvement := sides.buyOrder.After.Price.Minus(execution.Price); improvement > 0 {
		if err := sides.buyerParty.account.ReleaseReservation(improvement.Times(execution.Quantity)); err != nil {
			return model.Trade{}, err
		}
	}

	sellerAccountBefore := sides.sellerParty.account
	if err := sides.sellerParty.account.Credit(amount); err != nil {
		return model.Trade{}, err
	}

	sellerPositionBefore := sides.sellerParty.position
	if _, err := sides.sellerParty.position.DeliverReservedTokens(execution.Quantity); err != nil {
		return model.Trade{}, err
	}

	buyerPositionBefore := sides.buyerParty.position
	if err := sides.buyerParty.position.Receive(execution.Quantity, amount); err != nil {
		return model.Trade{}, err
	}

	return session.SettleExecution(ctx, repository.ExecutionSettlement{
		Actor:          command.Actor,
		BuyOrder:       sides.buyOrder,
		SellOrder:      sides.sellOrder,
		BuyerPosition:  repository.PositionChange{Before: buyerPositionBefore, After: sides.buyerParty.position},
		SellerPosition: repository.PositionChange{Before: sellerPositionBefore, After: sides.sellerParty.position},
		BuyerAccount:   repository.AccountChange{Before: buyerAccountBefore, After: sides.buyerParty.account},
		SellerAccount:  repository.AccountChange{Before: sellerAccountBefore, After: sides.sellerParty.account},
		Quantity:       execution.Quantity,
		Price:          execution.Price,
	})
}

func (s *MarketService) applyRemainder(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	submitter *tradingParty,
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
		return s.cancelRemainder(ctx, session, command, incoming, submitter, remaining)
	}

	return fmt.Errorf("vigencia desconocida: %s", incoming.TimeInForce)
}

func (s *MarketService) cancelRemainder(
	ctx context.Context,
	session *repository.MarketSession,
	command PlaceOrderCommand,
	incoming *model.Order,
	submitter *tradingParty,
	remaining int64,
) error {
	if _, err := s.releaseReservation(ctx, session, command.Actor, incoming, submitter, remaining); err != nil {
		return err
	}

	orderBefore := *incoming
	if err := incoming.Cancel(s.now()); err != nil {
		return err
	}

	orderChange := repository.OrderChange{Before: orderBefore, After: *incoming}

	return session.SaveOrder(ctx, command.Actor, repository.ActionCancel, orderChange)
}

type releasedReservation struct {
	credits model.Credits
	tokens  int64
}

func (s *MarketService) releaseReservation(
	ctx context.Context,
	session *repository.MarketSession,
	actor string,
	order *model.Order,
	holder *tradingParty,
	remaining int64,
) (releasedReservation, error) {
	if order.Side == model.SideBuy {
		credits := order.Price.Times(remaining)

		before := holder.account
		if err := holder.account.ReleaseReservation(credits); err != nil {
			return releasedReservation{}, err
		}

		change := repository.AccountChange{Before: before, After: holder.account}
		if err := session.SaveAccount(ctx, actor, repository.ActionRelease, change); err != nil {
			return releasedReservation{}, err
		}

		return releasedReservation{credits: credits}, nil
	}

	before := holder.position
	if err := holder.position.ReleaseReservedTokens(remaining); err != nil {
		return releasedReservation{}, err
	}

	change := repository.PositionChange{Before: before, After: holder.position}
	if err := session.SavePosition(ctx, actor, repository.ActionRelease, change); err != nil {
		return releasedReservation{}, err
	}

	return releasedReservation{tokens: remaining}, nil
}

type CancelOrderCommand struct {
	Actor   string
	UserID  int64
	OrderID int64
}

type CancelledOrder struct {
	Order           model.Order
	ReleasedCredits model.Credits
	ReleasedTokens  int64
}

func (s *MarketService) CancelOrder(ctx context.Context, command CancelOrderCommand) (CancelledOrder, error) {
	known, err := s.market.Order(ctx, command.OrderID)
	if err != nil {
		return CancelledOrder{}, translateMarketError(err)
	}
	if known.UserID != command.UserID {
		return CancelledOrder{}, ErrNotOwner
	}

	var cancelled CancelledOrder

	err = s.market.WithPlayerLocked(ctx, known.PlayerID, func(ctx context.Context, session *repository.MarketSession) error {
		result, cancelErr := s.cancelWithinLock(ctx, session, command)
		if cancelErr != nil {
			return cancelErr
		}
		cancelled = result

		return nil
	})
	if err != nil {
		return CancelledOrder{}, translateMarketError(err)
	}

	return cancelled, nil
}

func (s *MarketService) cancelWithinLock(
	ctx context.Context,
	session *repository.MarketSession,
	command CancelOrderCommand,
) (CancelledOrder, error) {
	order, err := session.Order(ctx, command.OrderID)
	if err != nil {
		return CancelledOrder{}, err
	}
	if order.UserID != command.UserID {
		return CancelledOrder{}, ErrNotOwner
	}
	if !order.IsLive() {
		return CancelledOrder{}, model.ErrOrderNotLive
	}

	holder, err := loadParty(ctx, session, order.UserID, order.PlayerID)
	if err != nil {
		return CancelledOrder{}, err
	}

	released, err := s.releaseReservation(ctx, session, command.Actor, &order, holder, order.RemainingQuantity())
	if err != nil {
		return CancelledOrder{}, err
	}

	before := order
	if cancelErr := order.Cancel(s.now()); cancelErr != nil {
		return CancelledOrder{}, cancelErr
	}

	change := repository.OrderChange{Before: before, After: order}
	if saveErr := session.SaveOrder(ctx, command.Actor, repository.ActionCancel, change); saveErr != nil {
		return CancelledOrder{}, saveErr
	}

	return CancelledOrder{
		Order:           order,
		ReleasedCredits: released.credits,
		ReleasedTokens:  released.tokens,
	}, nil
}

func (s *MarketService) ListOpenOrders(ctx context.Context, userID int64, playerID *int64) ([]model.Order, error) {
	return s.market.LiveOrdersOf(ctx, userID, playerID)
}

func translateMarketError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, model.ErrOrderNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrPlayerInactive):
		return ErrPlayerInactive
	default:
		return err
	}
}
