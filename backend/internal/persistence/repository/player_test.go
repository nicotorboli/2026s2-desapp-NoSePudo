package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type mockPlayerSql struct {
	err error
	players []model.Player
}

func (m *mockPlayerSql) GetPlayer(ctx context.Context) ([]model.Player, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.players, nil
}

func TestPlayerRepository_GetPlayer_Success(t *testing.T) {
	expected := []model.Player{
		{ID: 1, Name: "Ernesto Provitillo", Position: 5},
	}
	mockSql := &mockPlayerSql{players: expected}
	repo := repository.NewPlayerRepository(mockSql)

	got, err := repo.GetPlayer(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(got) != 1 || got[0] != expected[0] {
		t.Fatalf("expected %+v, got %+v", expected, got)
	}
}

func TestPlayerRepository_GetPlayer_Error(t *testing.T) {
	expectedErr := errors.New("sql failure")
	mockSql := &mockPlayerSql{err: expectedErr}
	repo := repository.NewPlayerRepository(mockSql)

	_, err := repo.GetPlayer(t.Context())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

