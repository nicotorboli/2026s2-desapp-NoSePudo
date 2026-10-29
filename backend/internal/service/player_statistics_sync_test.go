package service_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/whoscored"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

// startStatisticsPostgres levanta un Postgres real sembrado con db/init.sql
// para que el servicio corra contra persistencia real y no contra un mock.
func startStatisticsPostgres(t *testing.T) *sql.DB {
	t.Helper()
	ctx := t.Context()

	container, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "db", "init.sql")),
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
		t.Fatalf("no se pudo levantar Postgres con testcontainers: %v", err)
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

var serviceSeedSeq int64

// seedPlayer inserta un jugador activo y devuelve su id.
func seedPlayer(t *testing.T, db *sql.DB, name, club, code string) int64 {
	t.Helper()
	serviceSeedSeq++

	var id int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position, active)
		 VALUES ($1, $2, $3, 'Premier League', $4, 'Midfielder', TRUE) RETURNING id`,
		serviceSeedSeq, name, club, code,
	).Scan(&id); err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}
	return id
}

// fakeWhoscoredClient reemplaza al adaptador externo: es el único mock que
// admite la capa de servicio, porque la API de WhoScored no se toca en tests.
type fakeWhoscoredClient struct {
	rows map[string][]model.PlayerStatisticsRow
	errs map[string]error
}

func (f *fakeWhoscoredClient) FetchLeagueStatistics(ctx context.Context, code string) ([]model.PlayerStatisticsRow, error) {
	if f.errs != nil {
		if err, ok := f.errs[code]; ok {
			return nil, err
		}
	}
	if f.rows != nil {
		return f.rows[code], nil
	}
	return nil, nil
}

func newStatisticsSyncService(t *testing.T, db *sql.DB, adapter *fakeWhoscoredClient) *service.PlayerStatisticsSyncServiceImpl {
	t.Helper()
	daos := dao.NewContainer(db)
	repos := repository.NewContainer(db, daos)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewPlayerStatisticsSyncService(adapter, repos.Player, repos.PlayerStatistics, logger)
}

func TestSyncStatistics_MatchesAndPersists(t *testing.T) {
	db := startStatisticsPostgres(t)

	seedPlayer(t, db, "Bukayo Saka", "Arsenal FC", "PL")
	seedPlayer(t, db, "Erling Haaland", "Manchester City FC", "PL")
	seedPlayer(t, db, "Dominic Solanke", "AFC Bournemouth", "PL")

	adapter := &fakeWhoscoredClient{rows: map[string][]model.PlayerStatisticsRow{
		"PL": {
			{Name: "Bukayo Saka", ClubName: "Arsenal", Goals: 10, Assists: 5, Shots: 2.5, KeyPasses: 1.8, Dribbles: 2.1, Tackles: 1.1, Interceptions: 0.6, Fouls: 0.7, Clears: 0.4, Blocks: 0.2, WasDribbled: 0.9, Rating: 7.8},
			{Name: "Erling Haaland", ClubName: "Man City", Goals: 20, Rating: 8.5},
			{Name: "Dominic Solanke", ClubName: "Bournemouth", Goals: 9, Rating: 7.2},
			{Name: "Jugador Fantasma", ClubName: "Equipo Inexistente", Goals: 1, Rating: 6.0},
		},
	}}

	resp, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context())
	if err != nil {
		t.Fatalf("SyncStatistics: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("esperaba status success, obtuve %s", resp.Status)
	}
	if resp.TotalProcessed != 4 || resp.TotalMatched != 3 || resp.TotalUnmatched != 1 || resp.TotalInserted != 3 {
		t.Errorf("conteos inesperados: %+v", resp)
	}

	var count int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM player_statistics`).Scan(&count); err != nil {
		t.Fatalf("no se pudo contar las filas: %v", err)
	}
	if count != 3 {
		t.Errorf("esperaba 3 filas persistidas, obtuve %d", count)
	}
}

func TestSyncStatistics_UniqueNameFallback(t *testing.T) {
	db := startStatisticsPostgres(t)

	seedPlayer(t, db, "Lewis Dunk", "Brighton & Hove Albion FC", "PL")

	adapter := &fakeWhoscoredClient{rows: map[string][]model.PlayerStatisticsRow{
		"PL": {{Name: "Lewis Dunk", ClubName: "Brighton", Goals: 1, Rating: 6.9}},
	}}

	resp, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context())
	if err != nil {
		t.Fatalf("SyncStatistics: %v", err)
	}
	if resp.TotalMatched != 1 || resp.TotalUnmatched != 0 {
		t.Errorf("el fallback por nombre único debería matchear: %+v", resp)
	}
}

func TestSyncStatistics_DuplicateNameIsNotMatched(t *testing.T) {
	db := startStatisticsPostgres(t)

	seedPlayer(t, db, "Gabriel", "Arsenal FC", "PL")
	seedPlayer(t, db, "Gabriel", "Chelsea FC", "PL")

	adapter := &fakeWhoscoredClient{rows: map[string][]model.PlayerStatisticsRow{
		"PL": {{Name: "Gabriel", ClubName: "Equipo Desconocido", Goals: 1, Rating: 6.5}},
	}}

	resp, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context())
	if err != nil {
		t.Fatalf("SyncStatistics: %v", err)
	}
	if resp.TotalMatched != 0 || resp.TotalUnmatched != 1 {
		t.Errorf("un homónimo no debe producir matcheo falso: %+v", resp)
	}
}

func TestSyncStatistics_RateLimit(t *testing.T) {
	db := startStatisticsPostgres(t)

	adapter := &fakeWhoscoredClient{errs: map[string]error{"PL": whoscored.ErrRateLimitExceeded}}

	if _, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context()); !errors.Is(err, service.ErrStatisticsRateLimitExceeded) {
		t.Fatalf("esperaba ErrStatisticsRateLimitExceeded, obtuve %v", err)
	}
}

func TestSyncStatistics_BotChallenge(t *testing.T) {
	db := startStatisticsPostgres(t)

	adapter := &fakeWhoscoredClient{errs: map[string]error{"PL": whoscored.ErrBotChallenge}}

	if _, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context()); !errors.Is(err, service.ErrStatisticsProviderBlocked) {
		t.Fatalf("esperaba ErrStatisticsProviderBlocked, obtuve %v", err)
	}
}

func TestSyncStatistics_PartialSuccessWhenALeagueFails(t *testing.T) {
	db := startStatisticsPostgres(t)

	seedPlayer(t, db, "Bukayo Saka", "Arsenal FC", "PL")

	adapter := &fakeWhoscoredClient{
		rows: map[string][]model.PlayerStatisticsRow{
			"PL": {{Name: "Bukayo Saka", ClubName: "Arsenal", Goals: 10, Rating: 7.8}},
		},
		errs: map[string]error{"BL1": whoscored.ErrBotChallenge},
	}

	resp, err := newStatisticsSyncService(t, db, adapter).SyncStatistics(t.Context())
	if err != nil {
		t.Fatalf("SyncStatistics: %v", err)
	}
	if resp.Status != "partial_success" {
		t.Errorf("esperaba partial_success, obtuve %s", resp.Status)
	}
	if resp.TotalProcessed != 1 || resp.TotalMatched != 1 || resp.TotalInserted != 1 {
		t.Errorf("la liga completada debe quedar contabilizada: %+v", resp)
	}
}
