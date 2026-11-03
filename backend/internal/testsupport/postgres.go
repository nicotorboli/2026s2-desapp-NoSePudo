package testsupport

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	templateDatabase = "nsp_db"
	adminDatabase    = "postgres"
	databaseUser     = "devuser"
	databasePassword = "devpassword"
)

type SharedPostgres struct {
	container testcontainers.Container
	admin     *sql.DB
	host      string
	port      string
	counter   atomic.Int64
}

func StartSharedPostgres(ctx context.Context, initScript string) (*SharedPostgres, error) {
	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(initScript),
		postgres.WithDatabase(templateDatabase),
		postgres.WithUsername(databaseUser),
		postgres.WithPassword(databasePassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("levantar Postgres con testcontainers (¿Docker está corriendo?): %w", err)
	}

	shared := &SharedPostgres{container: container}
	if err = shared.resolveAddress(ctx); err != nil {
		return nil, err
	}

	shared.admin, err = sql.Open("postgres", shared.dataSourceFor(adminDatabase))
	if err != nil {
		return nil, fmt.Errorf("abrir la conexión de administración: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = shared.admin.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("la base no responde: %w", err)
	}

	return shared, nil
}

func (s *SharedPostgres) resolveAddress(ctx context.Context) error {
	host, err := s.container.Host(ctx)
	if err != nil {
		return fmt.Errorf("resolver el host del contenedor: %w", err)
	}

	mapped, err := s.container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return fmt.Errorf("resolver el puerto del contenedor: %w", err)
	}

	s.host = host
	s.port = mapped.Port()

	return nil
}

func (s *SharedPostgres) dataSourceFor(database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		databaseUser, databasePassword, s.host, s.port, database)
}

func (s *SharedPostgres) Close(ctx context.Context) {
	if s.admin != nil {
		_ = s.admin.Close()
	}
	if s.container != nil {
		_ = testcontainers.TerminateContainer(s.container)
	}
}

func (s *SharedPostgres) Fresh(t *testing.T) *sql.DB {
	t.Helper()

	name := fmt.Sprintf("prueba_%d", s.counter.Add(1))

	// PostgreSQL no acepta parámetros para identificadores, así que el nombre
	// se interpola; se cita con pq.QuoteIdentifier para que no sea inyectable.
	// NOSONAR: el nombre lo genera este helper y el template es una constante.
	createQuery := fmt.Sprintf(
		`CREATE DATABASE %s TEMPLATE %s`,
		pq.QuoteIdentifier(name),
		pq.QuoteIdentifier(templateDatabase),
	)

	if _, err := s.admin.ExecContext(t.Context(), createQuery); err != nil { // NOSONAR
		t.Fatalf("no se pudo clonar la base de pruebas: %v", err)
	}

	db, err := sql.Open("postgres", s.dataSourceFor(name))
	if err != nil {
		t.Fatalf("no se pudo abrir la base %s: %v", name, err)
	}

	t.Cleanup(func() {
		_ = db.Close()

		dropCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dropQuery := fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, pq.QuoteIdentifier(name))

		if _, dropErr := s.admin.ExecContext(dropCtx, dropQuery); dropErr != nil { // NOSONAR
			t.Logf("no se pudo descartar la base %s: %v", name, dropErr)
		}
	})

	return db
}
