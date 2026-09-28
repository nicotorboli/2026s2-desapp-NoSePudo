package e2e_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/server"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

const (
	// El secreto mide los 32 bytes que exige la configuración.
	testJWTSecret = "0123456789abcdef0123456789abcdef"

	// El costo mínimo de bcrypt. Lo que estos casos prueban es el recorrido
	// completo, no cuánto tarda en hashear.
	testBcryptCost = 4
)

// startPostgres levanta un Postgres real sembrado con el mismo db/init.sql que
// monta docker-compose.
//
// Está duplicado a propósito con el helper de internal/persistence/repository:
// son dos paquetes de test distintos y compartirlo obligaría a inventar un
// paquete de soporte sólo para cuarenta líneas.
func startPostgres(t *testing.T) *sql.DB {
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

// stack es la aplicación entera corriendo sobre una base real: el mismo
// armado de dependencias que hace cmd, servido por httptest.
type stack struct {
	db     *sql.DB
	server *httptest.Server
}

func newStack(t *testing.T) *stack {
	t.Helper()

	db := startPostgres(t)

	cfg := &configuration.Cfg{
		JWTSecret:  testJWTSecret,
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 168 * time.Hour,
		BcryptCost: testBcryptCost,
	}

	daos := dao.NewContainer(db)
	repos := repository.NewContainer(daos)
	adapterContainer := adapters.NewContainer(cfg)
	services := service.NewContainer(repos, adapterContainer)
	controllers := controller.NewContainer(services)
	middlewares := middleware.NewContainer()

	httpServer := httptest.NewServer(server.NewServer(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		controllers,
		middlewares,
	))
	t.Cleanup(httpServer.Close)

	return &stack{db: db, server: httpServer}
}

// countUsers es como se comprueba que una petición rechazada no dejó nada
// escrito.
func (s *stack) countUsers(t *testing.T) int {
	t.Helper()

	var count int
	if err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("no se pudieron contar las cuentas: %v", err)
	}

	return count
}
