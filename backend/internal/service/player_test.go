package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type mockPlayerRepository struct {
	listPlayersFn func(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error)
	getPlayerFn   func(ctx context.Context, id int64) (model.Player, error)
	savePlayersFn func(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error)
}

func (m *mockPlayerRepository) ListPlayers(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error) {
	if m.listPlayersFn != nil {
		return m.listPlayersFn(ctx, filter)
	}
	return model.PageResult[model.Player]{}, nil
}

func (m *mockPlayerRepository) GetPlayerByID(ctx context.Context, id int64) (model.Player, error) {
	if m.getPlayerFn != nil {
		return m.getPlayerFn(ctx, id)
	}
	return model.Player{}, nil
}

func (m *mockPlayerRepository) SavePlayers(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error) {
	if m.savePlayersFn != nil {
		return m.savePlayersFn(ctx, players, leagueCode, actor)
	}
	return len(players), 0, 0, nil
}

type mockFootballDataClient struct {
	fetchLeaguePlayersFn func(ctx context.Context, leagueCode string) ([]model.Player, error)
}

func (m *mockFootballDataClient) FetchLeaguePlayers(ctx context.Context, leagueCode string) ([]model.Player, error) {
	if m.fetchLeaguePlayersFn != nil {
		return m.fetchLeaguePlayersFn(ctx, leagueCode)
	}
	return nil, nil
}

func TestPlayerService_ListPlayers_Success(t *testing.T) {
	expectedPlayers := []model.Player{
		{ID: 1, Name: "Ernesto Provitillo", ClubName: "Arsenal", LeagueName: "Premier League", Position: "Midfielder"},
		{ID: 2, Name: "Alfre Montes de Oca", ClubName: "Chelsea", LeagueName: "Premier League", Position: "Goalkeeper"},
	}

	mockRepo := &mockPlayerRepository{
		listPlayersFn: func(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error) {
			return model.PageResult[model.Player]{
				Items:      expectedPlayers,
				Page:       1,
				Limit:      20,
				Total:      2,
				TotalPages: 1,
			}, nil
		},
	}

	svc := service.NewPlayerService(mockRepo)
	res, err := svc.ListPlayers(context.Background(), dto.PlayerFilterDTO{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Items) != 2 {
		t.Fatalf("expected 2 players, got %d", len(res.Items))
	}
	if res.Items[0].Name != "Ernesto Provitillo" || res.Items[0].Club != "Arsenal" || res.Items[0].League != "Premier League" || res.Items[0].Position != "Midfielder" {
		t.Errorf("unexpected player 0: %+v", res.Items[0])
	}
	if res.Items[1].Name != "Alfre Montes de Oca" || res.Items[1].Club != "Chelsea" || res.Items[1].League != "Premier League" || res.Items[1].Position != "Goalkeeper" {
		t.Errorf("unexpected player 1: %+v", res.Items[1])
	}
}

func TestPlayerService_GetPlayerByID_Success(t *testing.T) {
	dob := "1990-05-15"
	nat := "Argentina"
	num := 10
	now := time.Now().UTC()

	mockRepo := &mockPlayerRepository{
		getPlayerFn: func(ctx context.Context, id int64) (model.Player, error) {
			if id == 42 {
				return model.Player{
					ID:          42,
					ExternalID:  1234,
					Name:        "Test Player",
					ClubName:    "FC Barcelona",
					LeagueName:  "La Liga",
					LeagueCode:  "PD",
					Position:    "Attacker",
					DateOfBirth: &dob,
					Nationality: &nat,
					ShirtNumber: &num,
					Active:      true,
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			}
			return model.Player{}, model.ErrNotFound
		},
	}

	svc := service.NewPlayerService(mockRepo)
	res, err := svc.GetPlayerByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ID != 42 || res.ExternalID != 1234 || res.Name != "Test Player" || res.Club != "FC Barcelona" || res.LeagueCode != "PD" || res.Position != "Attacker" {
		t.Errorf("unexpected player detail: %+v", res)
	}
	if res.DateOfBirth == nil || *res.DateOfBirth != dob {
		t.Errorf("expected dob %s, got %v", dob, res.DateOfBirth)
	}
	if res.Nationality == nil || *res.Nationality != nat {
		t.Errorf("expected nat %s, got %v", nat, res.Nationality)
	}
	if res.ShirtNumber == nil || *res.ShirtNumber != num {
		t.Errorf("expected shirtNumber %d, got %v", num, res.ShirtNumber)
	}
}

func TestPlayerSyncService_SyncPlayers_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockAdapter := &mockFootballDataClient{
		fetchLeaguePlayersFn: func(ctx context.Context, leagueCode string) ([]model.Player, error) {
			return []model.Player{
				{ExternalID: 1, Name: "Player " + leagueCode},
			}, nil
		},
	}

	mockRepo := &mockPlayerRepository{
		savePlayersFn: func(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error) {
			return len(players), 1, 0, nil
		},
	}

	syncSvc := service.NewPlayerSyncService(mockAdapter, mockRepo, logger)
	resp, err := syncSvc.SyncPlayers(context.Background(), "system/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status success, got %s", resp.Status)
	}
	if resp.TotalProcessed != 5 {
		t.Errorf("expected 5 processed (1 per league), got %d", resp.TotalProcessed)
	}
	if resp.TotalUpdated != 5 {
		t.Errorf("expected 5 updated, got %d", resp.TotalUpdated)
	}
}

func TestPlayerSyncService_SyncPlayers_RateLimitHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	callCount := 0
	mockAdapter := &mockFootballDataClient{
		fetchLeaguePlayersFn: func(ctx context.Context, leagueCode string) ([]model.Player, error) {
			callCount++
			if callCount == 1 {
				return []model.Player{{ExternalID: 1, Name: "Player 1"}}, nil
			}
			return nil, model.ErrRateLimitExceeded
		},
	}

	mockRepo := &mockPlayerRepository{
		savePlayersFn: func(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error) {
			return len(players), 0, 0, nil
		},
	}

	syncSvc := service.NewPlayerSyncService(mockAdapter, mockRepo, logger)
	resp, err := syncSvc.SyncPlayers(context.Background(), "system/test")
	if err != nil {
		t.Fatalf("unexpected error on partial success: %v", err)
	}

	if resp.Status != "partial_success" {
		t.Errorf("expected status partial_success, got %s", resp.Status)
	}
	if resp.TotalProcessed != 1 {
		t.Errorf("expected 1 processed, got %d", resp.TotalProcessed)
	}
}

func TestPlayerSyncService_SyncPlayers_TotalRateLimitFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockAdapter := &mockFootballDataClient{
		fetchLeaguePlayersFn: func(ctx context.Context, leagueCode string) ([]model.Player, error) {
			return nil, model.ErrRateLimitExceeded
		},
	}
	mockRepo := &mockPlayerRepository{}

	syncSvc := service.NewPlayerSyncService(mockAdapter, mockRepo, logger)
	_, err := syncSvc.SyncPlayers(context.Background(), "system/test")
	if !errors.Is(err, model.ErrRateLimitExceeded) {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}
}
