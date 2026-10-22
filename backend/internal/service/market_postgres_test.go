package service_test

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
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

const marketFakeHash = "$2a$04$" + "abcdefghijklmnopqrstuv" + "wxyz0123456789ABCDEFGHIJKLMNOPQ"

var marketSeedCounter int64

type marketFixture struct {
	db      *sql.DB
	daos    *dao.Container
	market  *service.MarketService
	clock   time.Time
	players *dao.PlayerSql
}

func newMarketFixture(t *testing.T) *marketFixture {
	t.Helper()

	db := startMarketPostgres(t)
	daos := dao.NewContainer(db)
	repos := repository.NewContainer(db, daos)

	fixture := &marketFixture{
		db:      db,
		daos:    daos,
		clock:   time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		players: daos.Player,
	}
	fixture.market = service.NewMarketService(repos.Market, func() time.Time { return fixture.clock })

	return fixture
}

func startMarketPostgres(t *testing.T) *sql.DB {
	t.Helper()

	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "db", "init.sql")),
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

func (f *marketFixture) seedUser(t *testing.T, balance model.Credits) int64 {
	t.Helper()

	marketSeedCounter++

	var id int64
	err := f.db.QueryRowContext(t.Context(),
		`INSERT INTO users (email, password_hash, privilege, active) VALUES ($1, $2, $3, TRUE) RETURNING id`,
		fmt.Sprintf("mercado-%d@test.local", marketSeedCounter),
		marketFakeHash,
		int16(model.PrivilegeUser),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un usuario: %v", err)
	}

	if _, err = f.db.ExecContext(t.Context(),
		`INSERT INTO user_accounts (user_id, available_balance, reserved_balance) VALUES ($1, $2, 0)`,
		id, int64(balance),
	); err != nil {
		t.Fatalf("no se pudo sembrar la cuenta del usuario %d: %v", id, err)
	}

	return id
}

func (f *marketFixture) seedPlayer(t *testing.T) int64 {
	t.Helper()

	marketSeedCounter++

	var id int64
	err := f.db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES ($1, $2, 'Arsenal FC', 'Premier League', 'PL', 'Offence') RETURNING id`,
		marketSeedCounter,
		fmt.Sprintf("Jugador %d", marketSeedCounter),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}

	return id
}

func (f *marketFixture) deactivatePlayer(t *testing.T, playerID int64) {
	t.Helper()

	if _, err := f.db.ExecContext(t.Context(),
		`UPDATE players SET active = FALSE WHERE id = $1`, playerID,
	); err != nil {
		t.Fatalf("no se pudo desactivar el jugador %d: %v", playerID, err)
	}
}

func (f *marketFixture) seedQuote(t *testing.T, playerID int64, price model.Credits) {
	t.Helper()

	if _, err := f.db.ExecContext(t.Context(),
		`INSERT INTO player_quotes (player_id, price, strategy, strategy_version)
		 VALUES ($1, $2, 'initial', '1')`,
		playerID, int64(price),
	); err != nil {
		t.Fatalf("no se pudo sembrar la cotización del jugador %d: %v", playerID, err)
	}
}

func (f *marketFixture) seedRestingSell(t *testing.T, sellerID, playerID int64, price model.Credits, quantity int64) int64 {
	t.Helper()

	if _, err := f.db.ExecContext(t.Context(),
		`INSERT INTO user_positions (user_id, player_id, available_quantity, reserved_quantity, cost_basis)
		 VALUES ($1, $2, 0, $3, 0)
		 ON CONFLICT (user_id, player_id)
		 DO UPDATE SET reserved_quantity = user_positions.reserved_quantity + $3`,
		sellerID, playerID, quantity,
	); err != nil {
		t.Fatalf("no se pudo sembrar la posición del vendedor %d: %v", sellerID, err)
	}

	var orderID int64
	err := f.db.QueryRowContext(t.Context(),
		`INSERT INTO market_orders (user_id, player_id, side, time_in_force, status, price, quantity, created_at)
		 VALUES ($1, $2, 'SELL', 'GTC', 'PENDING', $3, $4, $5) RETURNING id`,
		sellerID, playerID, int64(price), quantity, f.clock,
	).Scan(&orderID)
	if err != nil {
		t.Fatalf("no se pudo sembrar la orden de venta: %v", err)
	}

	return orderID
}

func (f *marketFixture) account(t *testing.T, userID int64) model.Account {
	t.Helper()

	account, err := f.daos.Account.GetByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("no se pudo leer la cuenta del usuario %d: %v", userID, err)
	}

	return account
}

func (f *marketFixture) position(t *testing.T, userID, playerID int64) model.Position {
	t.Helper()

	positions, err := f.daos.Position.ListByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("no se pudieron leer las posiciones del usuario %d: %v", userID, err)
	}

	for _, position := range positions {
		if position.PlayerID == playerID {
			return position
		}
	}

	return model.Position{UserID: userID, PlayerID: playerID}
}

func (f *marketFixture) ledgerIsBalanced(t *testing.T) bool {
	t.Helper()

	var unbalanced int
	err := f.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM (
		     SELECT settlement_group FROM ledger_entries
		     GROUP BY settlement_group HAVING SUM(amount) <> 0
		 ) AS roto`,
	).Scan(&unbalanced)
	if err != nil {
		t.Fatalf("no se pudo verificar el balance del libro mayor: %v", err)
	}

	return unbalanced == 0
}
