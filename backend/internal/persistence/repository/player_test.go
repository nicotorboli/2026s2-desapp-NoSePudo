package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type mockPlayerDAO struct {
	listPlayersFn              func(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error)
	getPlayerByIDFn            func(ctx context.Context, id int64) (model.Player, error)
	getPlayerByExternalIDFn    func(ctx context.Context, externalID int64) (model.Player, error)
	upsertPlayerFn             func(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error)
	deactivateMissingPlayersFn func(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error)
}

func (m *mockPlayerDAO) ListPlayers(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error) {
	if m.listPlayersFn != nil {
		return m.listPlayersFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockPlayerDAO) GetPlayerByID(ctx context.Context, id int64) (model.Player, error) {
	if m.getPlayerByIDFn != nil {
		return m.getPlayerByIDFn(ctx, id)
	}
	return model.Player{}, nil
}

func (m *mockPlayerDAO) GetPlayerByExternalID(ctx context.Context, externalID int64) (model.Player, error) {
	if m.getPlayerByExternalIDFn != nil {
		return m.getPlayerByExternalIDFn(ctx, externalID)
	}
	return model.Player{}, nil
}

func (m *mockPlayerDAO) UpsertPlayer(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error) {
	if m.upsertPlayerFn != nil {
		return m.upsertPlayerFn(ctx, tx, p)
	}
	return 1, nil
}

func (m *mockPlayerDAO) DeactivateMissingPlayers(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error) {
	if m.deactivateMissingPlayersFn != nil {
		return m.deactivateMissingPlayersFn(ctx, tx, activeExternalIDs, leagueCode)
	}
	return nil, nil
}

type mockAuditDAO struct {
	insertAuditLogFn func(ctx context.Context, tx *sql.Tx, log model.AuditLog) error
}

func (m *mockAuditDAO) InsertAuditLog(ctx context.Context, tx *sql.Tx, log model.AuditLog) error {
	if m.insertAuditLogFn != nil {
		return m.insertAuditLogFn(ctx, tx, log)
	}
	return nil
}

func TestPlayerRepository_ListPlayers(t *testing.T) {
	mockPDAO := &mockPlayerDAO{
		listPlayersFn: func(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error) {
			return []model.Player{
				{ID: 1, Name: "Player 1", LeagueCode: "PL"},
				{ID: 2, Name: "Player 2", LeagueCode: "PL"},
			}, 25, nil
		},
	}
	mockADAO := &mockAuditDAO{}

	repo := repository.NewPlayerRepository(nil, mockPDAO, mockADAO)
	result, err := repo.ListPlayers(context.Background(), model.PlayerFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Total != 25 {
		t.Errorf("expected total 25, got %d", result.Total)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected totalPages 3, got %d", result.TotalPages)
	}
}

func TestPlayerRepository_GetPlayerByID(t *testing.T) {
	mockPDAO := &mockPlayerDAO{
		getPlayerByIDFn: func(ctx context.Context, id int64) (model.Player, error) {
			if id == 100 {
				return model.Player{ID: 100, Name: "Lionel Messi"}, nil
			}
			return model.Player{}, model.ErrNotFound
		},
	}
	mockADAO := &mockAuditDAO{}

	repo := repository.NewPlayerRepository(nil, mockPDAO, mockADAO)

	player, err := repo.GetPlayerByID(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.Name != "Lionel Messi" {
		t.Errorf("expected Lionel Messi, got %s", player.Name)
	}

	_, err = repo.GetPlayerByID(context.Background(), 999)
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
