package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var ErrNotFound = errors.New("player not found")

type PlayerDAO interface {
	ListPlayers(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error)
	GetPlayerByID(ctx context.Context, id int64) (model.Player, error)
	GetPlayerByExternalID(ctx context.Context, externalID int64) (model.Player, error)
	UpsertPlayer(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error)
	DeactivateMissingPlayers(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error)
}

type PlayerSql struct {
	db *sql.DB
}

func NewPlayerDao(db *sql.DB) *PlayerSql {
	return &PlayerSql{db: db}
}

// Los filtros viajan como predicados que se anulan solos: cuando el parámetro
// llega vacío la condición da TRUE y la consulta queda como si ese filtro no
// existiera. Así el SQL es un literal fijo en vez de texto armado en runtime
// (lo que Sonar marca como go:S2077), y las dos consultas comparten los mismos
// $1..$5 en el mismo orden.
const countPlayersQuery = `SELECT COUNT(*) FROM players
              WHERE ($1::bool OR active = TRUE)
                AND ($2::text = '' OR league_code = $2 OR league_name ILIKE $2)
                AND ($3::text = '' OR club_name ILIKE '%' || $3 || '%')
                AND ($4::text = '' OR position ILIKE '%' || $4 || '%')
                AND ($5::text = '' OR name ILIKE '%' || $5 || '%')`

const listPlayersQuery = `SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at
              FROM players
              WHERE ($1::bool OR active = TRUE)
                AND ($2::text = '' OR league_code = $2 OR league_name ILIKE $2)
                AND ($3::text = '' OR club_name ILIKE '%' || $3 || '%')
                AND ($4::text = '' OR position ILIKE '%' || $4 || '%')
                AND ($5::text = '' OR name ILIKE '%' || $5 || '%')
              ORDER BY name ASC
              LIMIT $6 OFFSET $7`

const getPlayerByIDQuery = `SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at
              FROM players WHERE id = $1`

const getPlayerByExternalIDQuery = `SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at
              FROM players WHERE external_id = $1`

const (
	defaultPage  = 1
	defaultLimit = 20
)

// rowScanner es lo único que las tres lecturas necesitan de *sql.Row y
// *sql.Rows, que no comparten interfaz en database/sql.
type rowScanner interface {
	Scan(dest ...any) error
}

// playerFilterArgs devuelve los cinco parámetros del filtro en el orden en que
// los esperan countPlayersQuery y listPlayersQuery.
func playerFilterArgs(f model.PlayerFilter) []any {
	return []any{f.IncludeInactive, f.League, f.Club, f.Position, f.Search}
}

func scanPlayer(scanner rowScanner) (model.Player, error) {
	var p model.Player
	err := scanner.Scan(
		&p.ID,
		&p.ExternalID,
		&p.Name,
		&p.ClubName,
		&p.LeagueName,
		&p.LeagueCode,
		&p.DateOfBirth,
		&p.Nationality,
		&p.Position,
		&p.ShirtNumber,
		&p.Active,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	return p, err
}

func (d *PlayerSql) ListPlayers(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error) {
	if d.db == nil {
		return nil, 0, fmt.Errorf("database connection is nil")
	}

	filterArgs := playerFilterArgs(f)

	var total int64
	if err := d.db.QueryRowContext(ctx, countPlayersQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count players: %w", err)
	}

	page := f.Page
	if page < defaultPage {
		page = defaultPage
	}
	limit := f.Limit
	if limit < 1 {
		limit = defaultLimit
	}
	offset := (page - 1) * limit

	rows, err := d.db.QueryContext(ctx, listPlayersQuery, append(filterArgs, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("query players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	players := make([]model.Player, 0, limit)
	for rows.Next() {
		p, scanErr := scanPlayer(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan player: %w", scanErr)
		}
		players = append(players, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return players, total, nil
}

func (d *PlayerSql) GetPlayerByID(ctx context.Context, id int64) (model.Player, error) {
	if d.db == nil {
		return model.Player{}, fmt.Errorf("database connection is nil")
	}

	p, err := scanPlayer(d.db.QueryRowContext(ctx, getPlayerByIDQuery, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Player{}, ErrNotFound
		}
		return model.Player{}, fmt.Errorf("get player by id: %w", err)
	}

	return p, nil
}

func (d *PlayerSql) GetPlayerByExternalID(ctx context.Context, externalID int64) (model.Player, error) {
	if d.db == nil {
		return model.Player{}, fmt.Errorf("database connection is nil")
	}

	p, err := scanPlayer(d.db.QueryRowContext(ctx, getPlayerByExternalIDQuery, externalID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Player{}, ErrNotFound
		}
		return model.Player{}, fmt.Errorf("get player by external id: %w", err)
	}

	return p, nil
}

// Las tres escrituras piden la transacción en vez de aceptar nil y caer a la
// conexión suelta: el upsert, la baja y su auditoría tienen que confirmarse
// juntos o no confirmarse, así que fuera de una transacción no hay caso de uso
// legítimo y se falla temprano.
func (d *PlayerSql) UpsertPlayer(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error) {
	if tx == nil {
		return 0, fmt.Errorf("upsert player requires a transaction")
	}

	query := `INSERT INTO players (external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
              ON CONFLICT (external_id) DO UPDATE SET
                name = EXCLUDED.name,
                club_name = EXCLUDED.club_name,
                league_name = EXCLUDED.league_name,
                league_code = EXCLUDED.league_code,
                date_of_birth = EXCLUDED.date_of_birth,
                nationality = EXCLUDED.nationality,
                position = EXCLUDED.position,
                shirt_number = EXCLUDED.shirt_number,
                active = EXCLUDED.active,
                updated_at = NOW()
              RETURNING id`

	var id int64
	err := tx.QueryRowContext(ctx, query,
		p.ExternalID, p.Name, p.ClubName, p.LeagueName, p.LeagueCode,
		p.DateOfBirth, p.Nationality, p.Position, p.ShirtNumber, p.Active,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert player: %w", err)
	}

	return id, nil
}

func (d *PlayerSql) DeactivateMissingPlayers(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error) {
	if tx == nil {
		return nil, fmt.Errorf("deactivate missing players requires a transaction")
	}

	var rows *sql.Rows
	var err error

	if len(activeExternalIDs) == 0 {
		query := `UPDATE players SET active = FALSE, updated_at = NOW() WHERE league_code = $1 AND active = TRUE RETURNING id`
		rows, err = tx.QueryContext(ctx, query, leagueCode)
	} else {
		query := `UPDATE players SET active = FALSE, updated_at = NOW() WHERE league_code = $1 AND active = TRUE AND NOT (external_id = ANY($2)) RETURNING id`
		rows, err = tx.QueryContext(ctx, query, leagueCode, pq.Array(activeExternalIDs))
	}

	if err != nil {
		return nil, fmt.Errorf("deactivate missing players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	deactivatedIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan deactivated player id: %w", err)
		}
		deactivatedIDs = append(deactivatedIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("deactivate missing players rows error: %w", err)
	}

	return deactivatedIDs, nil
}
