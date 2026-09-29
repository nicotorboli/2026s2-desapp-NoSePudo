package dao_test

import (
	"context"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestPlayerSql_NilDBHandling(t *testing.T) {
	playerDao := dao.NewPlayerDao(nil)
	ctx := context.Background()

	_, _, err := playerDao.ListPlayers(ctx, model.PlayerFilter{})
	if err == nil {
		t.Errorf("expected error with nil db, got nil")
	}

	_, err = playerDao.GetPlayerByID(ctx, 1)
	if err == nil {
		t.Errorf("expected error with nil db, got nil")
	}

	_, err = playerDao.GetPlayerByExternalID(ctx, 1)
	if err == nil {
		t.Errorf("expected error with nil db, got nil")
	}
}

func TestAuditSql_NilDBHandling(t *testing.T) {
	auditDao := dao.NewAuditDao(nil)
	ctx := context.Background()

	err := auditDao.InsertAuditLog(ctx, nil, model.AuditLog{
		EntityType: "player",
		EntityID:   1,
		Action:     "INSERT",
		Actor:      "test",
	})
	if err == nil {
		t.Errorf("expected error with nil db, got nil")
	}
}
