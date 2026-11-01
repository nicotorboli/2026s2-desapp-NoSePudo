package dao

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/lib/pq"
)

// SchemaSql responde una sola pregunta: cuáles de las tablas que el servicio
// necesita no están.
//
// Existe porque init.sql sólo corre cuando el volumen de Postgres está vacío,
// así que editarlo no cambia nada hasta recrear el volumen. Sin este chequeo
// el síntoma es un 500 con "relation \"users\" does not exist" tres capas más
// abajo, sin ninguna pista de que lo que cambió fue un archivo .sql que esa
// base nunca leyó.
type SchemaSql struct {
	db *sql.DB
}

func NewSchemaDao(db *sql.DB) *SchemaSql {
	return &SchemaSql{db: db}
}

// MissingTables devuelve, de las tablas pedidas, las que no existen en el
// esquema public, conservando el orden en que se pidieron.
func (d *SchemaSql) MissingTables(ctx context.Context, want []string) ([]string, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}
	if len(want) == 0 {
		return nil, nil
	}

	const query = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = ANY($1)`

	rows, err := d.db.QueryContext(ctx, query, pq.Array(want))
	if err != nil {
		return nil, fmt.Errorf("consultar el esquema: %w", err)
	}
	defer func() { _ = rows.Close() }()

	present := make([]string, 0, len(want))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("leer el nombre de la tabla: %w", err)
		}
		present = append(present, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer el esquema: %w", err)
	}

	missing := make([]string, 0)
	for _, name := range want {
		if !slices.Contains(present, name) {
			missing = append(missing, name)
		}
	}

	return missing, nil
}
