package dao_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func startPostgres(t *testing.T) *sql.DB {
	t.Helper()

	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "..", "db", "init.sql")),
		postgres.WithDatabase("nsp_db"),
		postgres.WithUsername("devuser"),
		postgres.WithPassword("devpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("no se pudo levantar Postgres con testcontainers (¿Docker está corriendo?): %v", err)
	}

	t.Cleanup(func() {
		if terminateErr := testcontainers.TerminateContainer(container); terminateErr != nil {
			t.Logf("no se pudo terminar el contenedor: %v", terminateErr)
		}
	})

	dataSource, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("no se pudo armar la cadena de conexión: %v", err)
	}

	db, err := sql.Open("postgres", dataSource)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("la base no responde: %v", err)
	}

	return db
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
