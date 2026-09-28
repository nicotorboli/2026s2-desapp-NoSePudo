package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerSql struct {
	Db *sql.DB
}

func (dao *PlayerSql) GetPlayer(ctx context.Context) ([]model.Player, error) {
	if dao.Db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := "SELECT id, name, position FROM players"
	rows, err := dao.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query players: %w", err)
	}
	defer func() { _ = rows.Close() }()

	players := make([]model.Player, 0)
	for rows.Next() {
		var p model.Player
		if err := rows.Scan(&p.ID, &p.Name, &p.Position); err != nil {
			return nil, fmt.Errorf("scan player: %w", err)
		}
		players = append(players, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return players, nil
}

func NewPlayerDao(db *sql.DB) *PlayerSql {
	return &PlayerSql{
		Db: db,
	}
}
