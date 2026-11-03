package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/testsupport"
)

// startPostgres levanta un Postgres real sembrado con el mismo db/init.sql que
// monta docker-compose, y devuelve la conexión.
//
// Es real y no un mock a propósito: lo único que un repositorio tiene es el
// SQL, así que mockear el DAO probaría la delegación y no la consulta. El
// índice único sobre email, por ejemplo, sólo se puede comprobar contra una
// base que lo tenga.
var sharedPostgres *testsupport.SharedPostgres

func TestMain(m *testing.M) {
	ctx := context.Background()

	shared, err := testsupport.StartSharedPostgres(ctx, filepath.Join("..", "..", "..", "db", "init.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo preparar Postgres para los tests de los repositorios: %v\n", err)
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
