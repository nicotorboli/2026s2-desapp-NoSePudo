package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (d *PlayerSql) ListPlayers(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error) {
	if d.db == nil {
		return nil, 0, fmt.Errorf("database connection is nil")
	}

	whereClauses := make([]string, 0)
	args := make([]any, 0)
	argIdx := 1

	if !f.IncludeInactive {
		whereClauses = append(whereClauses, "active = TRUE")
	}

	if f.League != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(league_code = $%d OR league_name ILIKE $%d)", argIdx, argIdx))
		args = append(args, f.League)
		argIdx++
	}

	if f.Club != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("club_name ILIKE $%d", argIdx))
		args = append(args, "%"+f.Club+"%")
		argIdx++
	}

	if f.Position != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("position ILIKE $%d", argIdx))
		args = append(args, "%"+f.Position+"%")
		argIdx++
	}

	if f.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Total count query
	countQuery := "SELECT COUNT(*) FROM players" + whereSQL
	var total int64
	if err := d.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count players: %w", err)
	}

	// 2. Paginated rows query
	page := f.Page
	if page < 1 {
		page = 1
	}
	limit := f.Limit
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	queryArgs := make([]any, len(args), len(args)+2)
	copy(queryArgs, args)
	queryArgs = append(queryArgs, limit, offset)
	dataQuery := fmt.Sprintf(
		"SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at FROM players%s ORDER BY name ASC LIMIT $%d OFFSET $%d",
		whereSQL, argIdx, argIdx+1,
	)

	rows, err := d.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	players := make([]model.Player, 0, limit)
	for rows.Next() {
		var p model.Player
		if err := rows.Scan(
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
		); err != nil {
			return nil, 0, fmt.Errorf("scan player: %w", err)
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

	query := `SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at 
              FROM players WHERE id = $1`

	var p model.Player
	err := d.db.QueryRowContext(ctx, query, id).Scan(
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

	query := `SELECT id, external_id, name, club_name, league_name, league_code, date_of_birth, nationality, position, shirt_number, active, created_at, updated_at 
              FROM players WHERE external_id = $1`

	var p model.Player
	err := d.db.QueryRowContext(ctx, query, externalID).Scan(
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
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Player{}, ErrNotFound
		}
		return model.Player{}, fmt.Errorf("get player by external id: %w", err)
	}

	return p, nil
}

func (d *PlayerSql) UpsertPlayer(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error) {
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
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, query,
			p.ExternalID, p.Name, p.ClubName, p.LeagueName, p.LeagueCode,
			p.DateOfBirth, p.Nationality, p.Position, p.ShirtNumber, p.Active,
		).Scan(&id)
	} else {
		err = d.db.QueryRowContext(ctx, query,
			p.ExternalID, p.Name, p.ClubName, p.LeagueName, p.LeagueCode,
			p.DateOfBirth, p.Nationality, p.Position, p.ShirtNumber, p.Active,
		).Scan(&id)
	}

	if err != nil {
		return 0, fmt.Errorf("upsert player: %w", err)
	}

	return id, nil
}

func (d *PlayerSql) DeactivateMissingPlayers(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error) {
	var rows *sql.Rows
	var err error

	if len(activeExternalIDs) == 0 {
		query := `UPDATE players SET active = FALSE, updated_at = NOW() WHERE league_code = $1 AND active = TRUE RETURNING id`
		if tx != nil {
			rows, err = tx.QueryContext(ctx, query, leagueCode)
		} else {
			rows, err = d.db.QueryContext(ctx, query, leagueCode)
		}
	} else {
		query := `UPDATE players SET active = FALSE, updated_at = NOW() WHERE league_code = $1 AND active = TRUE AND NOT (external_id = ANY($2)) RETURNING id`
		if tx != nil {
			rows, err = tx.QueryContext(ctx, query, leagueCode, pq.Array(activeExternalIDs))
		} else {
			rows, err = d.db.QueryContext(ctx, query, leagueCode, pq.Array(activeExternalIDs))
		}
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
