package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func newPlayerStatisticsService(t *testing.T, db *sql.DB) *service.PlayerStatisticsServiceImpl {
	t.Helper()
	daos := dao.NewContainer(db)
	repos := repository.NewContainer(db, daos)
	return service.NewPlayerStatisticsService(repos.Player, repos.PlayerStatistics)
}

func TestPlayerStatisticsService_GetPlayerStatistics_NewestFirst(t *testing.T) {
	db := startStatisticsPostgres(t)
	svc := newPlayerStatisticsService(t, db)

	playerID := seedPlayer(t, db, "Bukayo Saka", "Arsenal FC", "PL")

	daos := dao.NewContainer(db)
	repo := repository.NewPlayerStatisticsRepository(db, daos.PlayerStatistics)

	if _, _, err := repo.SaveStatistics(t.Context(), []model.PlayerStatistics{
		{PlayerID: playerID, Goals: 10, Rating: 7.8},
	}); err != nil {
		t.Fatalf("primera SaveStatistics: %v", err)
	}
	if _, _, err := repo.SaveStatistics(t.Context(), []model.PlayerStatistics{
		{PlayerID: playerID, Goals: 11, Rating: 8.1},
	}); err != nil {
		t.Fatalf("segunda SaveStatistics: %v", err)
	}

	items, err := svc.GetPlayerStatistics(t.Context(), playerID)
	if err != nil {
		t.Fatalf("GetPlayerStatistics: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("esperaba 2 snapshots, obtuve %d", len(items))
	}
	if items[0].Goals != 11 || items[0].Rating != 8.1 {
		t.Errorf("la más reciente debe ir primero: %+v", items[0])
	}
	if items[1].Goals != 10 || items[1].Rating != 7.8 {
		t.Errorf("la más vieja debe ir después: %+v", items[1])
	}
}

func TestPlayerStatisticsService_GetPlayerStatistics_PlayerNotFound(t *testing.T) {
	db := startStatisticsPostgres(t)
	svc := newPlayerStatisticsService(t, db)

	if _, err := svc.GetPlayerStatistics(t.Context(), 999999); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("esperaba ErrNotFound, obtuve %v", err)
	}
}

func TestPlayerStatisticsService_GetPlayerStatistics_EmptyWhenNoRows(t *testing.T) {
	db := startStatisticsPostgres(t)
	svc := newPlayerStatisticsService(t, db)

	playerID := seedPlayer(t, db, "Lewis Dunk", "Brighton & Hove Albion FC", "PL")

	items, err := svc.GetPlayerStatistics(t.Context(), playerID)
	if err != nil {
		t.Fatalf("GetPlayerStatistics: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("esperaba lista vacía, obtuve %d", len(items))
	}
}
