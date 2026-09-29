package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

// El sincronizador es el único que escribe jugadores, y siempre con este actor.
const testSyncActor = "system/sync"

// leagueNames evita arrastrar el nombre de la liga por cada caso: el código ya
// lo determina, igual que en los datos que llegan del proveedor.
var leagueNames = map[string]string{
	"PL": "Premier League",
	"PD": "La Liga",
}

// newPlayerRepositoryOnPostgres arma el repositorio con los DAOs reales sobre
// una base de verdad.
//
// Con el DAO mockeado sólo se puede probar la delegación, y lo que este
// repositorio realmente hace está en el SQL y en la transacción: el upsert por
// external_id, la baja de los que el proveedor dejó de mandar, la auditoría de
// cada cambio y el rollback cuando algo del camino falla.
func newPlayerRepositoryOnPostgres(t *testing.T) (*repository.PlayerRepositoryImpl, *sql.DB) {
	t.Helper()

	db := startPostgres(t)

	return repository.NewPlayerRepository(db, dao.NewPlayerDao(db), dao.NewAuditDao(db)), db
}

func samplePlayer(externalID int64, name, club, leagueCode, position string) model.Player {
	return model.Player{
		ExternalID: externalID,
		Name:       name,
		ClubName:   club,
		LeagueName: leagueNames[leagueCode],
		LeagueCode: leagueCode,
		Position:   position,
	}
}

func stringPtr(value string) *string { return &value }

func intPtr(value int) *int { return &value }

// auditEntry es una fila de la bitácora tal como quedó escrita.
type auditEntry struct {
	diffOld map[string]any
	diffNew map[string]any
	action  string
	actor   string
}

func auditEntriesFor(t *testing.T, db *sql.DB, entityID int64) []auditEntry {
	t.Helper()

	rows, err := db.QueryContext(t.Context(),
		`SELECT action, actor, diff_old, diff_new FROM player_audit_logs
		 WHERE entity_type = 'player' AND entity_id = $1 ORDER BY id`, entityID)
	if err != nil {
		t.Fatalf("no se pudo leer la bitácora: %v", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]auditEntry, 0)
	for rows.Next() {
		var entry auditEntry
		var diffOldJSON, diffNewJSON []byte
		if scanErr := rows.Scan(&entry.action, &entry.actor, &diffOldJSON, &diffNewJSON); scanErr != nil {
			t.Fatalf("no se pudo leer una entrada de la bitácora: %v", scanErr)
		}
		entry.diffOld = decodeDiff(t, diffOldJSON)
		entry.diffNew = decodeDiff(t, diffNewJSON)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("no se pudo recorrer la bitácora: %v", err)
	}

	return entries
}

func decodeDiff(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	if len(raw) == 0 {
		return nil
	}

	diff := make(map[string]any)
	if err := json.Unmarshal(raw, &diff); err != nil {
		t.Fatalf("el diff guardado no es un objeto JSON: %v (%s)", err, raw)
	}

	return diff
}

func playerIDOf(t *testing.T, db *sql.DB, externalID int64) int64 {
	t.Helper()

	var id int64
	if err := db.QueryRowContext(t.Context(),
		`SELECT id FROM players WHERE external_id = $1`, externalID).Scan(&id); err != nil {
		t.Fatalf("no se encontró el jugador %d: %v", externalID, err)
	}

	return id
}

func countPlayers(t *testing.T, db *sql.DB) int {
	t.Helper()

	var total int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM players`).Scan(&total); err != nil {
		t.Fatalf("no se pudieron contar los jugadores: %v", err)
	}

	return total
}

// Escenario: el proveedor manda jugadores que la base no tiene. Quedan
// guardados y cada alta deja su entrada en la bitácora.
func TestSavePlayersInsertsTheIncomingPlayersAndAuditsEachInsert(t *testing.T) {
	repo, db := newPlayerRepositoryOnPostgres(t)

	incoming := []model.Player{
		samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder"),
		samplePlayer(2, "Cole Palmer", "Chelsea FC", "PL", "Midfielder"),
	}

	processed, updated, deactivated, err := repo.SavePlayers(t.Context(), incoming, "PL", testSyncActor)
	if err != nil {
		t.Fatalf("el sync falló: %v", err)
	}

	if processed != 2 || updated != 0 || deactivated != 0 {
		t.Errorf("processed/updated/deactivated = %d/%d/%d, se esperaba 2/0/0", processed, updated, deactivated)
	}
	if total := countPlayers(t, db); total != 2 {
		t.Errorf("quedaron %d jugadores en la base, se esperaban 2", total)
	}

	stored, err := repo.GetPlayerByID(t.Context(), playerIDOf(t, db, 1))
	if err != nil {
		t.Fatalf("no se pudo recuperar al jugador guardado: %v", err)
	}
	if stored.Name != "Bukayo Saka" || stored.ClubName != "Arsenal FC" || !stored.Active {
		t.Errorf("el jugador se guardó distinto de lo que llegó: %+v", stored)
	}

	entries := auditEntriesFor(t, db, stored.ID)
	if len(entries) != 1 {
		t.Fatalf("la bitácora tiene %d entradas, se esperaba 1", len(entries))
	}
	if entries[0].action != "INSERT" || entries[0].actor != testSyncActor {
		t.Errorf("acción/actor = %s/%s, se esperaba INSERT/%s", entries[0].action, entries[0].actor, testSyncActor)
	}
	if entries[0].diffOld != nil {
		t.Errorf("un alta no tiene estado anterior, pero se guardó %v", entries[0].diffOld)
	}
	if entries[0].diffNew["name"] != "Bukayo Saka" {
		t.Errorf("el diff del alta no describe al jugador: %v", entries[0].diffNew)
	}
}

// Escenario: el jugador ya estaba y el proveedor lo manda cambiado. Se
// actualiza, y la bitácora guarda sólo los campos que efectivamente cambiaron.
func TestSavePlayersAuditsOnlyTheFieldsThatChanged(t *testing.T) {
	repo, db := newPlayerRepositoryOnPostgres(t)

	original := samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder")
	original.ShirtNumber = intPtr(7)
	if _, _, _, err := repo.SavePlayers(t.Context(), []model.Player{original}, "PL", testSyncActor); err != nil {
		t.Fatalf("el sync inicial falló: %v", err)
	}

	changed := original
	changed.ClubName = "Manchester City FC"
	changed.ShirtNumber = intPtr(10)
	changed.Nationality = stringPtr("England")

	_, updated, _, err := repo.SavePlayers(t.Context(), []model.Player{changed}, "PL", testSyncActor)
	if err != nil {
		t.Fatalf("el segundo sync falló: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, se esperaba 1", updated)
	}

	playerID := playerIDOf(t, db, 1)
	entries := auditEntriesFor(t, db, playerID)
	if len(entries) != 2 {
		t.Fatalf("la bitácora tiene %d entradas, se esperaban 2 (alta y modificación)", len(entries))
	}

	update := entries[1]
	if update.action != "UPDATE" {
		t.Errorf("acción = %s, se esperaba UPDATE", update.action)
	}
	if update.diffOld["clubName"] != "Arsenal FC" || update.diffNew["clubName"] != "Manchester City FC" {
		t.Errorf("el diff del club no refleja el cambio: %v -> %v", update.diffOld, update.diffNew)
	}
	if update.diffNew["shirtNumber"] != float64(10) || update.diffNew["nationality"] != "England" {
		t.Errorf("el diff no refleja los campos opcionales: %v", update.diffNew)
	}
	if _, present := update.diffNew["name"]; present {
		t.Errorf("el nombre no cambió, no tendría que estar en el diff: %v", update.diffNew)
	}

	stored, err := repo.GetPlayerByID(t.Context(), playerID)
	if err != nil {
		t.Fatalf("no se pudo recuperar al jugador actualizado: %v", err)
	}
	if stored.ClubName != "Manchester City FC" {
		t.Errorf("el club quedó en %q, se esperaba Manchester City FC", stored.ClubName)
	}
}

// Escenario: el proveedor manda exactamente lo que ya está guardado. No hay
// nada que contar como actualizado ni que agregar a la bitácora.
func TestSavePlayersDoesNotAuditWhenNothingChanged(t *testing.T) {
	repo, db := newPlayerRepositoryOnPostgres(t)

	unchanged := []model.Player{samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder")}
	if _, _, _, err := repo.SavePlayers(t.Context(), unchanged, "PL", testSyncActor); err != nil {
		t.Fatalf("el sync inicial falló: %v", err)
	}

	processed, updated, deactivated, err := repo.SavePlayers(t.Context(), unchanged, "PL", testSyncActor)
	if err != nil {
		t.Fatalf("el segundo sync falló: %v", err)
	}
	if processed != 1 || updated != 0 || deactivated != 0 {
		t.Errorf("processed/updated/deactivated = %d/%d/%d, se esperaba 1/0/0", processed, updated, deactivated)
	}

	if entries := auditEntriesFor(t, db, playerIDOf(t, db, 1)); len(entries) != 1 {
		t.Errorf("la bitácora tiene %d entradas, se esperaba sólo la del alta", len(entries))
	}
}

// Escenario: un jugador que estaba en la liga deja de venir en la respuesta del
// proveedor. Se da de baja con su entrada en la bitácora, y las demás ligas no
// se tocan.
func TestSavePlayersDeactivatesThePlayersMissingFromTheLeague(t *testing.T) {
	repo, db := newPlayerRepositoryOnPostgres(t)

	premier := []model.Player{
		samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder"),
		samplePlayer(2, "Cole Palmer", "Chelsea FC", "PL", "Midfielder"),
	}
	if _, _, _, err := repo.SavePlayers(t.Context(), premier, "PL", testSyncActor); err != nil {
		t.Fatalf("el sync de la Premier falló: %v", err)
	}
	laLiga := []model.Player{samplePlayer(3, "Aitor Fernández", "Athletic Club", "PD", "Goalkeeper")}
	if _, _, _, err := repo.SavePlayers(t.Context(), laLiga, "PD", testSyncActor); err != nil {
		t.Fatalf("el sync de La Liga falló: %v", err)
	}

	// La segunda corrida de la Premier ya no trae a Palmer.
	_, _, deactivated, err := repo.SavePlayers(t.Context(), premier[:1], "PL", testSyncActor)
	if err != nil {
		t.Fatalf("el segundo sync de la Premier falló: %v", err)
	}
	if deactivated != 1 {
		t.Fatalf("deactivated = %d, se esperaba 1", deactivated)
	}

	palmerID := playerIDOf(t, db, 2)
	palmer, err := repo.GetPlayerByID(t.Context(), palmerID)
	if err != nil {
		t.Fatalf("no se pudo recuperar al jugador dado de baja: %v", err)
	}
	if palmer.Active {
		t.Error("el jugador que dejó de venir quedó activo")
	}

	entries := auditEntriesFor(t, db, palmerID)
	if len(entries) != 2 || entries[1].action != "DEACTIVATE" {
		t.Errorf("la bitácora del jugador dado de baja quedó en %+v", entries)
	}

	spaniard, err := repo.GetPlayerByID(t.Context(), playerIDOf(t, db, 3))
	if err != nil {
		t.Fatalf("no se pudo recuperar al jugador de otra liga: %v", err)
	}
	if !spaniard.Active {
		t.Error("el sync de una liga dio de baja a un jugador de otra")
	}
}

// Escenario riesgoso: la auditoría falla en medio del sync. El invariante es
// que no quede ningún jugador escrito a medias, porque el upsert y su bitácora
// comparten transacción.
func TestSavePlayersRollsBackTheWholeSyncWhenAnAuditFails(t *testing.T) {
	db := startPostgres(t)
	failingAudit := &mockAuditDAO{
		insertAuditLogFn: func(_ context.Context, _ *sql.Tx, _ model.AuditLog) error {
			return errors.New("la bitácora no está disponible")
		},
	}
	repo := repository.NewPlayerRepository(db, dao.NewPlayerDao(db), failingAudit)

	incoming := []model.Player{samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder")}

	if _, _, _, err := repo.SavePlayers(t.Context(), incoming, "PL", testSyncActor); err == nil {
		t.Fatal("el sync terminó bien aunque la auditoría falló")
	}

	if total := countPlayers(t, db); total != 0 {
		t.Errorf("quedaron %d jugadores escritos, se esperaba que la transacción volviera atrás", total)
	}
}

// Escenario: el catálogo se pide con cada filtro que expone la API. Los casos
// comparten una sola base porque ninguno escribe.
func TestListPlayersFiltersPaginatesAndCounts(t *testing.T) {
	repo, _ := newPlayerRepositoryOnPostgres(t)

	premier := []model.Player{
		samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder"),
		samplePlayer(2, "Cole Palmer", "Chelsea FC", "PL", "Midfielder"),
	}
	if _, _, _, err := repo.SavePlayers(t.Context(), premier, "PL", testSyncActor); err != nil {
		t.Fatalf("el sync de la Premier falló: %v", err)
	}
	if _, _, _, err := repo.SavePlayers(t.Context(),
		[]model.Player{samplePlayer(3, "Aitor Fernández", "Athletic Club", "PD", "Goalkeeper")},
		"PD", testSyncActor); err != nil {
		t.Fatalf("el sync de La Liga falló: %v", err)
	}
	// Palmer deja de venir: queda inactivo y sirve para el filtro que los incluye.
	if _, _, _, err := repo.SavePlayers(t.Context(), premier[:1], "PL", testSyncActor); err != nil {
		t.Fatalf("el segundo sync de la Premier falló: %v", err)
	}

	// El orden de los campos lo pide fieldalignment, no la legibilidad.
	cases := []struct {
		wantNames []string
		name      string
		filter    model.PlayerFilter
		wantTotal int64
	}{
		{
			name:      "sin filtros sólo trae a los activos, ordenados por nombre",
			filter:    model.PlayerFilter{},
			wantNames: []string{"Aitor Fernández", "Bukayo Saka"},
			wantTotal: 2,
		},
		{
			name:      "con los inactivos incluidos aparece el que se dio de baja",
			filter:    model.PlayerFilter{IncludeInactive: true},
			wantNames: []string{"Aitor Fernández", "Bukayo Saka", "Cole Palmer"},
			wantTotal: 3,
		},
		{
			name:      "la liga se puede pedir por código",
			filter:    model.PlayerFilter{League: "PL"},
			wantNames: []string{"Bukayo Saka"},
			wantTotal: 1,
		},
		{
			name:      "la liga también se puede pedir por nombre",
			filter:    model.PlayerFilter{League: "La Liga"},
			wantNames: []string{"Aitor Fernández"},
			wantTotal: 1,
		},
		{
			name:      "el club matchea parcial y sin distinguir mayúsculas",
			filter:    model.PlayerFilter{Club: "arsen"},
			wantNames: []string{"Bukayo Saka"},
			wantTotal: 1,
		},
		{
			name:      "la posición matchea parcial",
			filter:    model.PlayerFilter{Position: "goal"},
			wantNames: []string{"Aitor Fernández"},
			wantTotal: 1,
		},
		{
			name:      "la búsqueda mira el nombre del jugador",
			filter:    model.PlayerFilter{Search: "saka"},
			wantNames: []string{"Bukayo Saka"},
			wantTotal: 1,
		},
		{
			name:      "los filtros se combinan entre sí",
			filter:    model.PlayerFilter{League: "PL", Position: "mid", IncludeInactive: true},
			wantNames: []string{"Bukayo Saka", "Cole Palmer"},
			wantTotal: 2,
		},
		{
			name:      "una búsqueda sin resultados devuelve una página vacía",
			filter:    model.PlayerFilter{Search: "Maradona"},
			wantNames: []string{},
			wantTotal: 0,
		},
		{
			name:      "la segunda página trae al siguiente jugador",
			filter:    model.PlayerFilter{IncludeInactive: true, Page: 2, Limit: 1},
			wantNames: []string{"Bukayo Saka"},
			wantTotal: 3,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			page, err := repo.ListPlayers(t.Context(), testCase.filter)
			if err != nil {
				t.Fatalf("el listado falló: %v", err)
			}

			if page.Total != testCase.wantTotal {
				t.Errorf("total = %d, se esperaba %d", page.Total, testCase.wantTotal)
			}

			names := make([]string, 0, len(page.Items))
			for _, player := range page.Items {
				names = append(names, player.Name)
			}
			if len(names) != len(testCase.wantNames) {
				t.Fatalf("la página trajo %v, se esperaba %v", names, testCase.wantNames)
			}
			for i, want := range testCase.wantNames {
				if names[i] != want {
					t.Errorf("items[%d] = %q, se esperaba %q", i, names[i], want)
				}
			}
		})
	}
}

// Escenario: sin página ni tamaño pedidos, el listado usa los valores por
// defecto y los informa, para que el cliente sepa qué página está viendo.
func TestListPlayersFallsBackToTheDefaultPageAndLimit(t *testing.T) {
	repo, _ := newPlayerRepositoryOnPostgres(t)

	if _, _, _, err := repo.SavePlayers(t.Context(),
		[]model.Player{samplePlayer(1, "Bukayo Saka", "Arsenal FC", "PL", "Midfielder")},
		"PL", testSyncActor); err != nil {
		t.Fatalf("el sync falló: %v", err)
	}

	page, err := repo.ListPlayers(t.Context(), model.PlayerFilter{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("el listado falló: %v", err)
	}

	if page.Page != 1 || page.Limit != 20 || page.TotalPages != 1 {
		t.Errorf("page/limit/totalPages = %d/%d/%d, se esperaba 1/20/1", page.Page, page.Limit, page.TotalPages)
	}
	if len(page.Items) != 1 {
		t.Errorf("la página trajo %d jugadores, se esperaba 1", len(page.Items))
	}
}

// Escenario: se pide un jugador que la base no tiene. El repositorio traduce el
// vacío del DAO a su propio ErrNotFound, que es lo que el controller mapea a
// 404.
func TestGetPlayerByIDTranslatesAnUnknownIDToNotFound(t *testing.T) {
	repo, _ := newPlayerRepositoryOnPostgres(t)

	if _, err := repo.GetPlayerByID(t.Context(), 999); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("error = %v, se esperaba repository.ErrNotFound", err)
	}
}
