package service

import (
	"context"
	"errors"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var (
	ErrP2PSelfTrade    = model.ErrP2PSessionSelfTrade
	ErrP2PExpired      = model.ErrP2PSessionExpired
	ErrP2PAlreadyTaken = model.ErrP2PSessionAlreadyTaken
	ErrP2PNotLive      = model.ErrP2PSessionNotLive
	ErrP2PNotFound     = model.ErrP2PSessionNotFound
	ErrP2PForbidden    = errors.New("sólo el vendedor puede cancelar su propia oferta")
)

type P2PRepositoryInterface interface {
	Create(ctx context.Context, session model.P2PSession, actor string) (model.P2PSession, error)
	GetByToken(ctx context.Context, token string) (model.P2PSession, error)
	Accept(ctx context.Context, token string, buyerID int64, now time.Time, actor string) (model.Trade, error)
	Cancel(ctx context.Context, token string, actorID int64, now time.Time, actor string) error
	ExpirePending(ctx context.Context, now time.Time) error
}

type P2PPlayerRepository interface {
	GetPlayerByID(ctx context.Context, id int64) (model.Player, error)
}

type CreateP2PCommand struct {
	Actor    string
	SellerID int64
	PlayerID int64
	Quantity int64
	Price    model.Credits
}

type P2PService struct {
	repo       P2PRepositoryInterface
	playerRepo P2PPlayerRepository
	now        func() time.Time
}

func NewP2PService(repo P2PRepositoryInterface, playerRepo P2PPlayerRepository, now func() time.Time) *P2PService {
	if now == nil {
		now = time.Now
	}
	return &P2PService{
		repo:       repo,
		playerRepo: playerRepo,
		now:        now,
	}
}

func (s *P2PService) Create(ctx context.Context, cmd CreateP2PCommand) (model.P2PSession, model.Player, error) {
	if cmd.Quantity <= 0 {
		return model.P2PSession{}, model.Player{}, ErrInvalidQuantity
	}
	if cmd.Price <= 0 {
		return model.P2PSession{}, model.Player{}, ErrInvalidPrice
	}

	player, err := s.playerRepo.GetPlayerByID(ctx, cmd.PlayerID)
	if err != nil {
		return model.P2PSession{}, model.Player{}, err
	}
	if !player.Active {
		return model.P2PSession{}, model.Player{}, ErrPlayerInactive
	}

	now := s.now()
	session := model.P2PSession{
		SellerID:  cmd.SellerID,
		PlayerID:  cmd.PlayerID,
		Quantity:  cmd.Quantity,
		Price:     cmd.Price,
		Status:    model.P2PStatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(5 * time.Minute),
	}

	created, err := s.repo.Create(ctx, session, cmd.Actor)
	if err != nil {
		return model.P2PSession{}, model.Player{}, err
	}

	return created, player, nil
}

func (s *P2PService) Get(ctx context.Context, token string) (model.P2PSession, model.Player, error) {
	session, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return model.P2PSession{}, model.Player{}, err
	}

	player, err := s.playerRepo.GetPlayerByID(ctx, session.PlayerID)
	if err != nil {
		return model.P2PSession{}, model.Player{}, err
	}

	return session, player, nil
}

func (s *P2PService) Accept(ctx context.Context, token string, buyerID int64, actor string) (model.Trade, error) {
	trade, err := s.repo.Accept(ctx, token, buyerID, s.now(), actor)
	if err != nil {
		return model.Trade{}, err
	}
	return trade, nil
}

func (s *P2PService) Cancel(ctx context.Context, token string, actorID int64, actor string) error {
	session, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return err
	}

	if session.SellerID != actorID {
		return ErrP2PForbidden
	}

	return s.repo.Cancel(ctx, token, actorID, s.now(), actor)
}
