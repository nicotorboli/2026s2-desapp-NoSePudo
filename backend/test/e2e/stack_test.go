package e2e_test

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
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
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
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
	auth   *service.Auth
	logs   *capturedLogs
}

// capturedLogs junta lo que el servidor escribe, para poder revisarlo después.
// Lleva mutex porque los eventos salen de las goroutines que atienden cada
// petición.
type capturedLogs struct {
	builder strings.Builder
	mutex   sync.Mutex
}

func (c *capturedLogs) Write(p []byte) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.builder.Write(p)
}

// lines devuelve los eventos emitidos hasta ahora, uno por línea.
func (c *capturedLogs) lines() []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	emitted := strings.TrimSpace(c.builder.String())
	if emitted == "" {
		return nil
	}

	return strings.Split(emitted, "\n")
}

func (c *capturedLogs) text() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.builder.String()
}

func newStack(t *testing.T) *stack {
	t.Helper()
	return newStackWith(t, 15*time.Minute, 168*time.Hour)
}

// newStackWithAccessTTL es para los casos que necesitan ver expirar una
// credencial de acceso de verdad, sin esperar quince minutos.
func newStackWithAccessTTL(t *testing.T, accessTTL time.Duration) *stack {
	t.Helper()
	return newStackWith(t, accessTTL, 168*time.Hour)
}

// newStackWithRefreshTTL hace lo mismo con la de renovación, que por defecto
// vive una semana.
func newStackWithRefreshTTL(t *testing.T, refreshTTL time.Duration) *stack {
	t.Helper()
	return newStackWith(t, 15*time.Minute, refreshTTL)
}

func newStackWith(t *testing.T, accessTTL, refreshTTL time.Duration) *stack {
	t.Helper()

	db := startPostgres(t)

	cfg := &configuration.Cfg{
		JWTSecret:  testJWTSecret,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
		BcryptCost: testBcryptCost,
	}

	daos := dao.NewContainer(db)
	repos := repository.NewContainer(daos)
	adapterContainer := adapters.NewContainer(cfg)
	services := service.NewContainer(repos, adapterContainer)
	controllers := controller.NewContainer(services)
	middlewares := middleware.NewContainer(adapterContainer.JWT)

	logs := &capturedLogs{}

	httpServer := httptest.NewServer(server.NewServer(
		slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		controllers,
		middlewares,
	))
	t.Cleanup(httpServer.Close)

	return &stack{db: db, server: httpServer, auth: services.Auth, logs: logs}
}

// ensureSuperuser aprovisiona el superusuario como lo hace cmd al arrancar.
func (s *stack) ensureSuperuser(t *testing.T, email, password string) {
	t.Helper()

	if err := s.auth.EnsureSuperuser(t.Context(), email, password); err != nil {
		t.Fatalf("no se pudo aprovisionar el superusuario: %v", err)
	}
}

// countSuperusers es como se comprueba el requerimiento: exactamente uno.
func (s *stack) countSuperusers(t *testing.T) int {
	t.Helper()

	var count int
	if err := s.db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM users WHERE privilege = $1", int16(model.PrivilegeSuperuser),
	).Scan(&count); err != nil {
		t.Fatalf("no se pudieron contar los superusuarios: %v", err)
	}

	return count
}

// countLiveRefreshTokens es como se comprueba que la respuesta al robo dejó la
// cuenta sin credenciales usables.
func (s *stack) countLiveRefreshTokens(t *testing.T) int {
	t.Helper()

	var count int
	if err := s.db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE used_at IS NULL AND revoked_at IS NULL AND expires_at > NOW()",
	).Scan(&count); err != nil {
		t.Fatalf("no se pudieron contar las credenciales vivas: %v", err)
	}

	return count
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
