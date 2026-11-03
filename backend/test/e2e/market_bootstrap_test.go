package e2e_test

import (
	"database/sql"
	"fmt"
	"testing"
)

func (s *stack) seedPlayers(t *testing.T, howMany int) []int64 {
	t.Helper()

	ids := make([]int64, 0, howMany)
	for i := range howMany {
		var id int64
		err := s.db.QueryRowContext(t.Context(),
			`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
			 VALUES ($1, $2, 'Arsenal FC', 'Premier League', 'PL', 'Offence') RETURNING id`,
			1000+i, fmt.Sprintf("Jugador %d", i+1),
		).Scan(&id)
		if err != nil {
			t.Fatalf("no se pudo sembrar el jugador %d: %v", i+1, err)
		}
		ids = append(ids, id)
	}

	return ids
}

func (s *stack) countRows(t *testing.T, query string, args ...any) int {
	t.Helper()

	var total int
	if err := s.db.QueryRowContext(t.Context(), query, args...).Scan(&total); err != nil {
		t.Fatalf("no se pudo contar con %q: %v", query, err)
	}

	return total
}

func (s *stack) tokensOfPlayer(t *testing.T, playerID int64) int64 {
	t.Helper()

	var total sql.NullInt64
	if err := s.db.QueryRowContext(t.Context(),
		`SELECT SUM(available_quantity + reserved_quantity) FROM user_positions WHERE player_id = $1`,
		playerID,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudieron sumar los tokens del jugador %d: %v", playerID, err)
	}

	return total.Int64
}

func (s *stack) unbalancedSettlements(t *testing.T) int {
	t.Helper()

	return s.countRows(t,
		`SELECT COUNT(*) FROM (
		     SELECT settlement_group FROM ledger_entries
		     GROUP BY settlement_group HAVING SUM(amount) <> 0
		 ) AS roto`)
}

func TestElArranqueDejaElMercadoListoYRepetirloNoDuplicaNada(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, "admin@nosepudo.test", "superclave-larga")
	players := stack.seedPlayers(t, 3)

	if err := stack.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el primer arranque devolvió %v", err)
	}

	assertMarketIsReady := func(t *testing.T, momento string) {
		t.Helper()

		for _, playerID := range players {
			if tokens := stack.tokensOfPlayer(t, playerID); tokens != 100 {
				t.Errorf("%s: el jugador %d tiene %d tokens, se esperaban exactamente 100", momento, playerID, tokens)
			}

			quotes := stack.countRows(t,
				`SELECT COUNT(*) FROM player_quotes WHERE player_id = $1 AND strategy = 'initial'`, playerID)
			if quotes != 1 {
				t.Errorf("%s: el jugador %d tiene %d cotizaciones iniciales, se esperaba 1", momento, playerID, quotes)
			}

			offers := stack.countRows(t,
				`SELECT COUNT(*) FROM market_orders
				 WHERE player_id = $1 AND market_maker AND status IN ('PENDING', 'PARTIALLY_FILLED')`, playerID)
			if offers != 1 {
				t.Errorf("%s: el jugador %d tiene %d ofertas vivas del creador de mercado, se esperaba 1", momento, playerID, offers)
			}
		}
	}

	assertMarketIsReady(t, "tras el primer arranque")

	if err := stack.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el segundo arranque devolvió %v", err)
	}
	if err := stack.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el tercer arranque devolvió %v", err)
	}

	assertMarketIsReady(t, "tras repetir el arranque dos veces")

	if unbalanced := stack.unbalancedSettlements(t); unbalanced != 0 {
		t.Errorf("hay %d liquidaciones que no suman cero", unbalanced)
	}
}
