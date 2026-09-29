package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type PlayerRepository interface {
	ListPlayers(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error)
	GetPlayerByID(ctx context.Context, id int64) (model.Player, error)
	SavePlayers(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error)
}

type PlayerRepositoryImpl struct {
	db        *sql.DB
	playerDAO dao.PlayerDAO
	auditDAO  dao.AuditDAO
}

func NewPlayerRepository(db *sql.DB, playerDAO dao.PlayerDAO, auditDAO dao.AuditDAO) *PlayerRepositoryImpl {
	return &PlayerRepositoryImpl{
		db:        db,
		playerDAO: playerDAO,
		auditDAO:  auditDAO,
	}
}

func (r *PlayerRepositoryImpl) ListPlayers(ctx context.Context, filter model.PlayerFilter) (model.PageResult[model.Player], error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	players, total, err := r.playerDAO.ListPlayers(ctx, filter)
	if err != nil {
		return model.PageResult[model.Player]{}, fmt.Errorf("repository list players: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return model.PageResult[model.Player]{
		Items:      players,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *PlayerRepositoryImpl) GetPlayerByID(ctx context.Context, id int64) (model.Player, error) {
	player, err := r.playerDAO.GetPlayerByID(ctx, id)
	if err != nil {
		return model.Player{}, err
	}
	return player, nil
}

func (r *PlayerRepositoryImpl) SavePlayers(ctx context.Context, players []model.Player, leagueCode, actor string) (int, int, int, error) {
	if r.db == nil {
		return 0, 0, 0, fmt.Errorf("database connection is nil")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	processedCount := len(players)
	updatedCount := 0
	activeExternalIDs := make([]int64, 0, len(players))

	for _, p := range players {
		activeExternalIDs = append(activeExternalIDs, p.ExternalID)

		existingPlayer, getErr := r.playerDAO.GetPlayerByExternalID(ctx, p.ExternalID)
		isNew := false
		if getErr != nil {
			if errors.Is(getErr, model.ErrNotFound) {
				isNew = true
			} else {
				return 0, 0, 0, fmt.Errorf("check existing player %d: %w", p.ExternalID, getErr)
			}
		}

		p.Active = true
		playerID, upsertErr := r.playerDAO.UpsertPlayer(ctx, tx, p)
		if upsertErr != nil {
			return 0, 0, 0, fmt.Errorf("upsert player %d: %w", p.ExternalID, upsertErr)
		}

		if isNew {
			diffNew := map[string]any{
				"name":        p.Name,
				"clubName":    p.ClubName,
				"leagueName":  p.LeagueName,
				"leagueCode":  p.LeagueCode,
				"position":    p.Position,
				"dateOfBirth": p.DateOfBirth,
				"nationality": p.Nationality,
				"shirtNumber": p.ShirtNumber,
				"active":      true,
			}
			auditLog := model.AuditLog{
				EntityType: "player",
				EntityID:   playerID,
				Action:     "INSERT",
				Actor:      actor,
				DiffOld:    nil,
				DiffNew:    diffNew,
			}
			if auditErr := r.auditDAO.InsertAuditLog(ctx, tx, auditLog); auditErr != nil {
				return 0, 0, 0, fmt.Errorf("insert audit log for new player %d: %w", playerID, auditErr)
			}
		} else {
			diffOld, diffNew := computePlayerDiff(existingPlayer, p)
			if len(diffNew) > 0 {
				updatedCount++
				auditLog := model.AuditLog{
					EntityType: "player",
					EntityID:   playerID,
					Action:     "UPDATE",
					Actor:      actor,
					DiffOld:    diffOld,
					DiffNew:    diffNew,
				}
				if auditErr := r.auditDAO.InsertAuditLog(ctx, tx, auditLog); auditErr != nil {
					return 0, 0, 0, fmt.Errorf("insert audit log for updated player %d: %w", playerID, auditErr)
				}
			}
		}
	}

	deactivatedIDs, deactErr := r.playerDAO.DeactivateMissingPlayers(ctx, tx, activeExternalIDs, leagueCode)
	if deactErr != nil {
		return 0, 0, 0, fmt.Errorf("deactivate missing players: %w", deactErr)
	}

	for _, id := range deactivatedIDs {
		auditLog := model.AuditLog{
			EntityType: "player",
			EntityID:   id,
			Action:     "DEACTIVATE",
			Actor:      actor,
			DiffOld:    map[string]any{"active": true},
			DiffNew:    map[string]any{"active": false},
		}
		if auditErr := r.auditDAO.InsertAuditLog(ctx, tx, auditLog); auditErr != nil {
			return 0, 0, 0, fmt.Errorf("insert audit log for deactivated player %d: %w", id, auditErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, 0, fmt.Errorf("commit transaction: %w", err)
	}

	return processedCount, updatedCount, len(deactivatedIDs), nil
}

func computePlayerDiff(oldP, newP model.Player) (map[string]any, map[string]any) {
	diffOld := make(map[string]any)
	diffNew := make(map[string]any)

	if oldP.Name != newP.Name {
		diffOld["name"] = oldP.Name
		diffNew["name"] = newP.Name
	}
	if oldP.ClubName != newP.ClubName {
		diffOld["clubName"] = oldP.ClubName
		diffNew["clubName"] = newP.ClubName
	}
	if oldP.LeagueName != newP.LeagueName {
		diffOld["leagueName"] = oldP.LeagueName
		diffNew["leagueName"] = newP.LeagueName
	}
	if oldP.LeagueCode != newP.LeagueCode {
		diffOld["leagueCode"] = oldP.LeagueCode
		diffNew["leagueCode"] = newP.LeagueCode
	}
	if oldP.Position != newP.Position {
		diffOld["position"] = oldP.Position
		diffNew["position"] = newP.Position
	}
	if !equalStringPtr(oldP.DateOfBirth, newP.DateOfBirth) {
		diffOld["dateOfBirth"] = oldP.DateOfBirth
		diffNew["dateOfBirth"] = newP.DateOfBirth
	}
	if !equalStringPtr(oldP.Nationality, newP.Nationality) {
		diffOld["nationality"] = oldP.Nationality
		diffNew["nationality"] = newP.Nationality
	}
	if !equalIntPtr(oldP.ShirtNumber, newP.ShirtNumber) {
		diffOld["shirtNumber"] = oldP.ShirtNumber
		diffNew["shirtNumber"] = newP.ShirtNumber
	}
	if oldP.Active != newP.Active {
		diffOld["active"] = oldP.Active
		diffNew["active"] = newP.Active
	}

	return diffOld, diffNew
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
