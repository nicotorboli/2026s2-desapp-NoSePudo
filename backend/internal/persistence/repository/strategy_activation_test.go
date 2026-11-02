package repository_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func newActivationRepo(t *testing.T) (*repository.StrategyActivationRepositoryImpl, int64) {
	t.Helper()
	db := startPostgres(t)

	user, err := repository.NewUserRepository(dao.NewUserDao(db)).Insert(t.Context(), aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	return repository.NewStrategyActivationRepository(dao.NewStrategyActivationDao(db)), user.ID
}

func TestStrategyActivationActivateAndActive(t *testing.T) {
	repo, userID := newActivationRepo(t)
	ctx := t.Context()

	if err := repo.Activate(ctx, 1, userID); err != nil {
		t.Fatalf("Activate devolvió error: %v", err)
	}

	active, found, err := repo.Active(ctx)
	if err != nil {
		t.Fatalf("Active devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Active = false después de activar")
	}
	if active.StrategyID != 1 {
		t.Errorf("StrategyID = %d, want 1", active.StrategyID)
	}
}

func TestStrategyActivationActiveReportsLatest(t *testing.T) {
	repo, userID := newActivationRepo(t)
	ctx := t.Context()

	if err := repo.Activate(ctx, 1, userID); err != nil {
		t.Fatalf("Activate 1: %v", err)
	}
	if err := repo.Activate(ctx, 2, userID); err != nil {
		t.Fatalf("Activate 2: %v", err)
	}

	active, found, err := repo.Active(ctx)
	if err != nil {
		t.Fatalf("Active devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Active = false")
	}
	if active.StrategyID != 2 {
		t.Errorf("StrategyID = %d, want 2", active.StrategyID)
	}
}

func TestStrategyActivationActiveWithNone(t *testing.T) {
	repo, _ := newActivationRepo(t)

	if _, found, err := repo.Active(t.Context()); err != nil {
		t.Fatalf("Active devolvió error: %v", err)
	} else if found {
		t.Error("Active = true sin activaciones")
	}
}
