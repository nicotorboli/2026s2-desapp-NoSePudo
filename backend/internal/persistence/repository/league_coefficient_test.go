package repository_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func newCoefficientRepo(t *testing.T) (*repository.LeagueCoefficientRepositoryImpl, int64) {
	t.Helper()
	db := startPostgres(t)

	user, err := repository.NewUserRepository(dao.NewUserDao(db)).Insert(t.Context(), aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	return repository.NewLeagueCoefficientRepository(dao.NewLeagueCoefficientDao(db)), user.ID
}

func TestLeagueCoefficientSetAndCurrent(t *testing.T) {
	repo, userID := newCoefficientRepo(t)
	ctx := t.Context()

	if err := repo.Set(ctx, "PL", 1.5, userID); err != nil {
		t.Fatalf("Set devolvió error: %v", err)
	}

	current, found, err := repo.Current(ctx, "PL")
	if err != nil {
		t.Fatalf("Current devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Current = false después de Set")
	}
	if current.Coefficient != 1.5 {
		t.Errorf("Coefficient = %v, want 1.5", current.Coefficient)
	}
}

func TestLeagueCoefficientCurrentReportsLatest(t *testing.T) {
	repo, userID := newCoefficientRepo(t)
	ctx := t.Context()

	if err := repo.Set(ctx, "PL", 1.0, userID); err != nil {
		t.Fatalf("Set 1: %v", err)
	}
	if err := repo.Set(ctx, "PL", 0.5, userID); err != nil {
		t.Fatalf("Set 2: %v", err)
	}

	current, found, err := repo.Current(ctx, "PL")
	if err != nil {
		t.Fatalf("Current devolvió error: %v", err)
	}
	if !found {
		t.Fatal("Current = false")
	}
	if current.Coefficient != 0.5 {
		t.Errorf("Coefficient = %v, want 0.5", current.Coefficient)
	}
}

func TestLeagueCoefficientCurrentWithNone(t *testing.T) {
	repo, _ := newCoefficientRepo(t)

	if _, found, err := repo.Current(t.Context(), "PL"); err != nil {
		t.Fatalf("Current devolvió error: %v", err)
	} else if found {
		t.Error("Current = true sin coeficiente")
	}
}
