package dao_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/testsupport"
)

var sharedPostgres *testsupport.SharedPostgres

func TestMain(m *testing.M) {
	ctx := context.Background()

	shared, err := testsupport.StartSharedPostgres(ctx, filepath.Join("..", "..", "..", "db", "init.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo preparar Postgres para los tests del dao: %v\n", err)
		os.Exit(1)
	}
	sharedPostgres = shared

	code := m.Run()

	shared.Close(ctx)
	os.Exit(code)
}

func startPostgres(t *testing.T) *sql.DB {
	t.Helper()

	return sharedPostgres.Fresh(t)
}

const fakeBcryptHash = "$2a$04$" + "abcdefghijklmnopqrstuv" + "wxyz0123456789ABCDEFGHIJKLMNOPQ"

var seedCounter int64

func seedUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	seedCounter++

	var id int64
	err := db.QueryRowContext(t.Context(),
		`INSERT INTO users (email, password_hash, privilege, active) VALUES ($1, $2, $3, TRUE) RETURNING id`,
		fmt.Sprintf("usuario-%d@test.local", seedCounter),
		fakeBcryptHash,
		int16(model.PrivilegeUser),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un usuario: %v", err)
	}

	return id
}

func seedPlayer(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	seedCounter++

	var id int64
	err := db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES ($1, $2, 'Arsenal FC', 'Premier League', 'PL', 'Offence') RETURNING id`,
		seedCounter,
		fmt.Sprintf("Jugador %d", seedCounter),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}

	return id
}

func beginTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("no se pudo abrir la transacción: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return tx
}
