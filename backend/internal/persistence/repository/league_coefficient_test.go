package repository_test

import (
	"database/sql"
	"testing"
	"time"

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

func newCoefficientRepoWithDB(t *testing.T) (*repository.LeagueCoefficientRepositoryImpl, *sql.DB, int64) {
	t.Helper()
	db := startPostgres(t)

	user, err := repository.NewUserRepository(dao.NewUserDao(db)).Insert(t.Context(), aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	return repository.NewLeagueCoefficientRepository(dao.NewLeagueCoefficientDao(db)), db, user.ID
}

func seedCoefficientAt(t *testing.T, db *sql.DB, leagueCode string, coefficient float64, effectiveFrom time.Time, userID int64) {
	t.Helper()

	if _, err := db.ExecContext(t.Context(),
		`INSERT INTO league_coefficients (league_code, coefficient, effective_from, set_by) VALUES ($1, $2, $3, $4)`,
		leagueCode, coefficient, effectiveFrom, userID); err != nil {
		t.Fatalf("no se pudo sembrar el coeficiente: %v", err)
	}
}

func TestLeagueCoefficientAtOrBeforeDevuelveElVigente(t *testing.T) {
	repo, db, userID := newCoefficientRepoWithDB(t)
	primero := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	segundo := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	seedCoefficientAt(t, db, "PL", 1.5, primero, userID)
	seedCoefficientAt(t, db, "PL", 0.5, segundo, userID)

	antes, found, err := repo.AtOrBefore(t.Context(), "PL", primero.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("AtOrBefore devolvió error: %v", err)
	}
	if !found || antes.Coefficient != 1.5 {
		t.Errorf("coeficiente antes del segundo = %v (found=%v), se esperaba 1.5", antes.Coefficient, found)
	}

	despues, found, err := repo.AtOrBefore(t.Context(), "PL", segundo.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("AtOrBefore devolvió error: %v", err)
	}
	if !found || despues.Coefficient != 0.5 {
		t.Errorf("coeficiente después del segundo = %v (found=%v), se esperaba 0.5", despues.Coefficient, found)
	}
}

func TestLeagueCoefficientAtOrBeforeIncluyeElLimite(t *testing.T) {
	repo, db, userID := newCoefficientRepoWithDB(t)
	momento := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	seedCoefficientAt(t, db, "PL", 0.75, momento, userID)

	got, found, err := repo.AtOrBefore(t.Context(), "PL", momento)
	if err != nil {
		t.Fatalf("AtOrBefore devolvió error: %v", err)
	}
	if !found || got.Coefficient != 0.75 {
		t.Errorf("coeficiente en el límite = %v (found=%v), se esperaba 0.75", got.Coefficient, found)
	}
}

func TestLeagueCoefficientAtOrBeforeSinCoeficiente(t *testing.T) {
	repo, db, userID := newCoefficientRepoWithDB(t)
	momento := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	seedCoefficientAt(t, db, "PL", 1.5, momento, userID)

	if _, found, err := repo.AtOrBefore(t.Context(), "PL", momento.Add(-time.Hour)); err != nil {
		t.Fatalf("AtOrBefore devolvió error: %v", err)
	} else if found {
		t.Error("AtOrBefore encontró un coeficiente anterior a su vigencia")
	}
}
