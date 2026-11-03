package service

import (
	"context"
	"errors"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var ErrForbidden = errors.New("no se puede consultar el patrimonio de otro usuario")

type TransactionEntry struct {
	ExecutedAt  time.Time
	PlayerName  string
	Side        model.OrderSide
	ID          int64
	PlayerID    int64
	Quantity    int64
	Price       model.Credits
	TotalAmount model.Credits
}

type PortfolioService struct {
	portfolio *repository.PortfolioRepository
}

func NewPortfolioService(portfolio *repository.PortfolioRepository) *PortfolioService {
	return &PortfolioService{portfolio: portfolio}
}

func authorizeReadOf(requester model.User, targetUserID int64) error {
	if requester.ID == targetUserID || requester.Privilege == model.PrivilegeSuperuser {
		return nil
	}

	return ErrForbidden
}

func (s *PortfolioService) Portfolio(ctx context.Context, requester model.User, targetUserID int64) (model.Portfolio, error) {
	if err := authorizeReadOf(requester, targetUserID); err != nil {
		return model.Portfolio{}, err
	}

	account, err := s.portfolio.Account(ctx, targetUserID)
	if err != nil {
		return model.Portfolio{}, err
	}

	positions, err := s.portfolio.Positions(ctx, targetUserID)
	if err != nil {
		return model.Portfolio{}, err
	}

	playerIDs := playerIDsOf(positions)

	quotes, err := s.portfolio.LatestPrices(ctx, playerIDs)
	if err != nil {
		return model.Portfolio{}, err
	}

	players, err := s.portfolio.Players(ctx, playerIDs)
	if err != nil {
		return model.Portfolio{}, err
	}

	return model.BuildPortfolio(targetUserID, account, positions, quotes, players), nil
}

func playerIDsOf(positions []model.Position) []int64 {
	ids := make([]int64, 0, len(positions))
	for i := range positions {
		ids = append(ids, positions[i].PlayerID)
	}

	return ids
}

func (s *PortfolioService) Transactions(
	ctx context.Context,
	requester model.User,
	targetUserID int64,
	page, limit int,
) (model.PageResult[TransactionEntry], error) {
	if err := authorizeReadOf(requester, targetUserID); err != nil {
		return model.PageResult[TransactionEntry]{}, err
	}

	trades, err := s.portfolio.Trades(ctx, targetUserID, page, limit)
	if err != nil {
		return model.PageResult[TransactionEntry]{}, err
	}

	players, err := s.portfolio.Players(ctx, playerIDsOfTrades(trades.Items))
	if err != nil {
		return model.PageResult[TransactionEntry]{}, err
	}

	return model.PageResult[TransactionEntry]{
		Items:      entriesOf(trades.Items, targetUserID, players),
		Total:      trades.Total,
		Page:       trades.Page,
		Limit:      trades.Limit,
		TotalPages: trades.TotalPages,
	}, nil
}

func playerIDsOfTrades(trades []model.Trade) []int64 {
	ids := make([]int64, 0, len(trades))
	for i := range trades {
		ids = append(ids, trades[i].PlayerID)
	}

	return ids
}

func entriesOf(trades []model.Trade, userID int64, players map[int64]model.Player) []TransactionEntry {
	entries := make([]TransactionEntry, 0, len(trades))

	for i := range trades {
		side := model.SideSell
		if trades[i].BuyerID == userID {
			side = model.SideBuy
		}

		entries = append(entries, TransactionEntry{
			ExecutedAt:  trades[i].ExecutedAt,
			PlayerName:  players[trades[i].PlayerID].Name,
			Side:        side,
			ID:          trades[i].ID,
			PlayerID:    trades[i].PlayerID,
			Quantity:    trades[i].Quantity,
			Price:       trades[i].Price,
			TotalAmount: trades[i].TotalAmount,
		})
	}

	return entries
}
