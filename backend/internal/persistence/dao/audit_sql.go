package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type AuditDAO interface {
	InsertAuditLog(ctx context.Context, tx *sql.Tx, log model.AuditLog) error
}

type AuditSql struct {
	db *sql.DB
}

func NewAuditDao(db *sql.DB) *AuditSql {
	return &AuditSql{db: db}
}

func (d *AuditSql) InsertAuditLog(ctx context.Context, tx *sql.Tx, log model.AuditLog) error {
	// La auditoría es parte de la misma transacción que la escritura que
	// describe: sin transacción no hay nada que auditar de forma consistente.
	if tx == nil {
		return fmt.Errorf("insert audit log requires a transaction")
	}

	var diffOldJSON []byte
	var diffNewJSON []byte
	var err error

	if log.DiffOld != nil {
		diffOldJSON, err = json.Marshal(log.DiffOld)
		if err != nil {
			return fmt.Errorf("marshal diff_old: %w", err)
		}
	}

	if log.DiffNew != nil {
		diffNewJSON, err = json.Marshal(log.DiffNew)
		if err != nil {
			return fmt.Errorf("marshal diff_new: %w", err)
		}
	} else {
		diffNewJSON = []byte("{}")
	}

	query := `INSERT INTO player_audit_logs (entity_type, entity_id, action, actor, diff_old, diff_new, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	_, execErr := tx.ExecContext(ctx, query, log.EntityType, log.EntityID, log.Action, log.Actor, diffOldJSON, diffNewJSON)
	if execErr != nil {
		return fmt.Errorf("insert audit log: %w", execErr)
	}

	return nil
}
