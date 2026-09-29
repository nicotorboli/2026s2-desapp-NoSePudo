package service

import (
	"context"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type PlayerService interface {
	ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error)
	GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error)
}

type PlayerServiceImpl struct {
	repo repository.PlayerRepository
}

func NewPlayerService(repo repository.PlayerRepository) *PlayerServiceImpl {
	return &PlayerServiceImpl{repo: repo}
}

func (s *PlayerServiceImpl) ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
	modelFilter := dto.PlayerFilterAModelo(filter)
	pageResult, err := s.repo.ListPlayers(ctx, modelFilter)
	if err != nil {
		return dto.PaginatedResponse[dto.PlayerListItemResponse]{}, fmt.Errorf("service list players: %w", err)
	}

	items := make([]dto.PlayerListItemResponse, len(pageResult.Items))
	for i, p := range pageResult.Items {
		items[i] = dto.PlayerListItemDesdeModelo(p)
	}

	return dto.PaginatedResponse[dto.PlayerListItemResponse]{
		Items:      items,
		Page:       pageResult.Page,
		Limit:      pageResult.Limit,
		Total:      pageResult.Total,
		TotalPages: pageResult.TotalPages,
	}, nil
}

func (s *PlayerServiceImpl) GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
	player, err := s.repo.GetPlayerByID(ctx, id)
	if err != nil {
		return dto.PlayerDetailResponse{}, err
	}

	return dto.PlayerDetailDesdeModelo(player), nil
}
