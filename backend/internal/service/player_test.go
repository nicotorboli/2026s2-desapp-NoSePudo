package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type mockPlayerRepository struct {
	err     error
	players []model.Player
}

func (m *mockPlayerRepository) GetPlayer(ctx context.Context) ([]model.Player, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.players, nil
}

func TestPlayerService_ListPlayers_Success(t *testing.T) {
	expectedPlayers := []model.Player{
		{ID: 1, Name: "Ernesto Provitillo", Position: 5},
		{ID: 2, Name: "Alfre Montes de Oca", Position: 1},
	}
	mockRepo := &mockPlayerRepository{players: expectedPlayers}
	svc := service.NewPlayerService(mockRepo)

	players, err := svc.ListPlayers(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(players) != len(expectedPlayers) {
		t.Fatalf("expected %d players, got %d", len(expectedPlayers), len(players))
	}

	for i := range players {
		if players[i] != expectedPlayers[i] {
			t.Errorf("expected player %+v, got %+v", expectedPlayers[i], players[i])
		}
	}
}

func TestPlayerService_ListPlayers_Error(t *testing.T) {
	expectedErr := errors.New("db connection failure")
	mockRepo := &mockPlayerRepository{err: expectedErr}
	svc := service.NewPlayerService(mockRepo)

	_, err := svc.ListPlayers(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
