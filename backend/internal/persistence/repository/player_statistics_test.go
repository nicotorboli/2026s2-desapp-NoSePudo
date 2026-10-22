package repository_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

// startStatisticsPostgres levanta un Postgres real sembrado con db/init.sql y
// falla en voz alta si Docker no está disponible, como exige el AGENTS.md.
func startStatisticsPostgres(t *testing.T) *sql.DB {
	t.Helper()
	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "..", "db", "init.sql")),
		postgres.WithDatabase("nsp_db"),
		postgres.WithUsername("devuser"),
		postgres.WithPassword("devpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("no se pudo levantar Postgres con testcontainers (¿Docker está corriendo?): %v", err)
	}

	t.Cleanup(func() {
		if terminateErr := testcontainers.TerminateContainer(container); terminateErr != nil {
			t.Logf("no se pudo terminar el contenedor: %v", terminateErr)
		}
	})

	dataSource, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("no se pudo armar la cadena de conexión: %v", err)
	}

	db, err := sql.Open("postgres", dataSource)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("la base no responde: %v", err)
	}

	return db
}

var statsSeedSeq int64

// seedStatisticsPlayer inserta un jugador activo y devuelve su id.
func seedStatisticsPlayer(t *testing.T, db *sql.DB, name, club, code string) int64 {
	t.Helper()
	statsSeedSeq++

	var id int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position, active)
		 VALUES ($1, $2, $3, 'Premier League', $4, 'Midfielder', TRUE) RETURNING id`,
		statsSeedSeq, name, club, code,
	).Scan(&id); err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}
	return id
}

func TestPlayerStatisticsRepository_SaveAndList(t *testing.T) {
	db := startStatisticsPostgres(t)
	daos := dao.NewContainer(db)
	repo := repository.NewPlayerStatisticsRepository(db, daos.PlayerStatistics)

	playerID := seedStatisticsPlayer(t, db, "Bukayo Saka", "Arsenal FC", "PL")

	first := model.PlayerStatistics{
		PlayerID: playerID, Goals: 10, Assists: 5,
		Shots: 2.5, KeyPasses: 1.8, Dribbles: 2.1, Tackles: 1.1,
		Interceptions: 0.6, Fouls: 0.7, Clears: 0.4, Blocks: 0.2, WasDribbled: 0.9, Rating: 7.8,
	}

	processed, inserted, err := repo.SaveStatistics(t.Context(), []model.PlayerStatistics{first})
	if err != nil {
		t.Fatalf("SaveStatistics: %v", err)
	}
	if processed != 1 || inserted != 1 {
		t.Errorf("esperaba (1,1), obtuve (%d,%d)", processed, inserted)
	}

	// Segunda pasada: snapshot nuevo, nunca se actualiza el anterior.
	second := model.PlayerStatistics{
		PlayerID: playerID, Goals: 11, Assists: 6,
		Shots: 2.8, KeyPasses: 2.0, Dribbles: 2.3, Tackles: 1.2,
		Interceptions: 0.7, Fouls: 0.8, Clears: 0.5, Blocks: 0.3, WasDribbled: 1.0, Rating: 8.1,
	}
	if _, _, err = repo.SaveStatistics(t.Context(), []model.PlayerStatistics{second}); err != nil {
		t.Fatalf("segunda SaveStatistics: %v", err)
	}

	rows, err := repo.GetPlayerStatistics(t.Context(), playerID)
	if err != nil {
		t.Fatalf("GetPlayerStatistics: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("esperaba 2 snapshots, obtuve %d", len(rows))
	}
	if rows[0].Goals != 11 || rows[0].Rating != 8.1 {
		t.Errorf("la más reciente debe ir primero: %+v", rows[0])
	}
	if rows[1].Goals != 10 || rows[1].Rating != 7.8 {
		t.Errorf("la más vieja debe ir después: %+v", rows[1])
	}
}

func TestPlayerStatisticsRepository_ListsOnlyRequestedPlayer(t *testing.T) {
	db := startStatisticsPostgres(t)
	daos := dao.NewContainer(db)
	repo := repository.NewPlayerStatisticsRepository(db, daos.PlayerStatistics)

	saka := seedStatisticsPlayer(t, db, "Bukayo Saka", "Arsenal FC", "PL")
	gabriel := seedStatisticsPlayer(t, db, "Gabriel Magalhaes", "Arsenal FC", "PL")

	if _, _, err := repo.SaveStatistics(t.Context(), []model.PlayerStatistics{
		{PlayerID: saka, Goals: 10, Rating: 7.8},
		{PlayerID: gabriel, Goals: 2, Rating: 7.1},
	}); err != nil {
		t.Fatalf("SaveStatistics: %v", err)
	}

	rows, err := repo.GetPlayerStatistics(t.Context(), saka)
	if err != nil {
		t.Fatalf("GetPlayerStatistics: %v", err)
	}
	if len(rows) != 1 || rows[0].Goals != 10 {
		t.Errorf("esperaba solo las filas del jugador pedido: %+v", rows)
	}
}
