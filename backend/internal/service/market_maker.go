package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

const (
	TokensPerPlayer   = 100
	initialQuoteUnits = 1
	initialStrategy   = "initial"
	initialVersion    = "1"
	marketMakerActor  = "market-maker"
	scoreQuoteVersion = "1"
)

type MarketMakerService struct {
	market *repository.MarketRepository
	now    func() time.Time
}

func NewMarketMakerService(market *repository.MarketRepository, now func() time.Time) *MarketMakerService {
	return &MarketMakerService{market: market, now: now}
}

func (s *MarketMakerService) Bootstrap(ctx context.Context) error {
	superuser, err := s.market.MarketMaker(ctx)
	if err != nil {
		return fmt.Errorf("el creador de mercado necesita un superusuario: %w", err)
	}

	playerIDs, err := s.market.ActivePlayerIDs(ctx)
	if err != nil {
		return fmt.Errorf("listar los jugadores activos: %w", err)
	}

	for _, playerID := range playerIDs {
		if bootstrapErr := s.bootstrapPlayer(ctx, superuser.ID, playerID); bootstrapErr != nil {
			return fmt.Errorf("preparar el mercado del jugador %d: %w", playerID, bootstrapErr)
		}
	}

	return nil
}

func (s *MarketMakerService) bootstrapPlayer(ctx context.Context, superuserID, playerID int64) error {
	return s.market.WithPlayerLocked(ctx, playerID, func(ctx context.Context, session *repository.MarketSession) error {
		if _, err := session.InsertInitialQuote(ctx, playerID, model.CreditsFromUnits(initialQuoteUnits)); err != nil {
			return err
		}

		if _, err := session.CreatePositionIfAbsent(ctx, model.Position{
			UserID:            superuserID,
			PlayerID:          playerID,
			AvailableQuantity: TokensPerPlayer,
		}); err != nil {
			return err
		}

		return s.restUnsoldInventory(ctx, session, superuserID, playerID)
	})
}

func (s *MarketMakerService) restUnsoldInventory(
	ctx context.Context,
	session *repository.MarketSession,
	superuserID, playerID int64,
) error {
	if _, err := session.LiveMarketMakerOrder(ctx, playerID, model.SideSell); err == nil {
		return nil
	} else if !errors.Is(err, model.ErrOrderNotFound) {
		return err
	}

	position, err := session.LockPosition(ctx, superuserID, playerID)
	if err != nil {
		return err
	}
	if position.AvailableQuantity == 0 {
		return nil
	}

	quote, err := session.LatestQuote(ctx, playerID)
	if err != nil {
		return err
	}

	return s.restAtQuote(ctx, session, &position, quote.Price, position.AvailableQuantity)
}

func (s *MarketMakerService) restAtQuote(
	ctx context.Context,
	session *repository.MarketSession,
	position *model.Position,
	price model.Credits,
	quantity int64,
) error {
	before := *position
	if err := position.ReserveTokens(quantity); err != nil {
		return err
	}

	change := repository.PositionChange{Before: before, After: *position}
	if err := session.SavePosition(ctx, marketMakerActor, repository.ActionReserve, change); err != nil {
		return err
	}

	_, err := session.InsertOrder(ctx, marketMakerActor, model.Order{
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
		Side:        model.SideSell,
		TimeInForce: model.GoodTillCancelled,
		Status:      model.StatusPending,
		UserID:      position.UserID,
		PlayerID:    position.PlayerID,
		Quantity:    quantity,
		Price:       price,
		MarketMaker: true,
	})

	return err
}

func (s *MarketMakerService) RecordQuote(ctx context.Context, playerID int64, price model.Credits, strategy, version string) error {
	return s.recordQuote(ctx, model.PlayerQuote{
		CreatedAt:       s.now(),
		Strategy:        strategy,
		StrategyVersion: version,
		PlayerID:        playerID,
		Price:           price,
	})
}

func (s *MarketMakerService) RecordScoreQuote(ctx context.Context, playerID int64, score float64, scoreID *int64, strategyName string) error {
	return s.recordQuote(ctx, model.PlayerQuote{
		ScoreID:         scoreID,
		CreatedAt:       s.now(),
		Strategy:        strategyName,
		StrategyVersion: scoreQuoteVersion,
		PlayerID:        playerID,
		Price:           model.DefaultPricePolicy().Price(score),
	})
}

func (s *MarketMakerService) recordQuote(ctx context.Context, quote model.PlayerQuote) error {
	superuser, err := s.market.MarketMaker(ctx)
	if err != nil {
		return fmt.Errorf("el creador de mercado necesita un superusuario: %w", err)
	}

	return s.market.WithPlayerLocked(ctx, quote.PlayerID, func(ctx context.Context, session *repository.MarketSession) error {
		if _, recordErr := session.RecordQuote(ctx, quote); recordErr != nil {
			return recordErr
		}

		return s.repriceInventory(ctx, session, superuser.ID, quote.PlayerID, quote.Price)
	})
}

func (s *MarketMakerService) repriceInventory(
	ctx context.Context,
	session *repository.MarketSession,
	superuserID, playerID int64,
	price model.Credits,
) error {
	position, err := session.LockPosition(ctx, superuserID, playerID)
	if err != nil {
		if errors.Is(err, model.ErrPositionNotFound) {
			return nil
		}

		return err
	}

	resting, err := session.LiveMarketMakerOrder(ctx, playerID, model.SideSell)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			return s.restWhatIsAvailable(ctx, session, &position, price)
		}

		return err
	}

	if err = s.cancelRestingInventory(ctx, session, &position, resting); err != nil {
		return err
	}

	return s.restWhatIsAvailable(ctx, session, &position, price)
}

func (s *MarketMakerService) cancelRestingInventory(
	ctx context.Context,
	session *repository.MarketSession,
	position *model.Position,
	resting model.Order,
) error {
	remaining := resting.RemainingQuantity()

	before := *position
	if err := position.ReleaseReservedTokens(remaining); err != nil {
		return err
	}

	change := repository.PositionChange{Before: before, After: *position}
	if err := session.SavePosition(ctx, marketMakerActor, repository.ActionRelease, change); err != nil {
		return err
	}

	orderBefore := resting
	if err := resting.Cancel(s.now()); err != nil {
		return err
	}

	orderChange := repository.OrderChange{Before: orderBefore, After: resting}

	return session.SaveOrder(ctx, marketMakerActor, repository.ActionCancel, orderChange)
}

func (s *MarketMakerService) restWhatIsAvailable(
	ctx context.Context,
	session *repository.MarketSession,
	position *model.Position,
	price model.Credits,
) error {
	if position.AvailableQuantity == 0 {
		return nil
	}

	return s.restAtQuote(ctx, session, position, price, position.AvailableQuantity)
}
