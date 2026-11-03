package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func (s *stack) put(t *testing.T, path, authorization string, payload any) (int, string) {
	t.Helper()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("no se pudo serializar el cuerpo: %v", err)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, s.server.URL+path, bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response, err := s.server.Client().Do(request)
	if err != nil {
		t.Fatalf("la petición falló: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}

	return response.StatusCode, string(body)
}

func (s *stack) seedStatistics(t *testing.T, playerID int64, rating float64) {
	t.Helper()

	if _, err := s.db.ExecContext(t.Context(),
		`INSERT INTO player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, interceptions, fouls, clears, blocks, was_dribbled, rating)
		 VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, $2)`, playerID, rating); err != nil {
		t.Fatalf("no se pudieron sembrar las estadísticas del jugador %d: %v", playerID, err)
	}
}

func TestCotizacionFlujoCompleto(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, "admin@nosepudo.test", "superclave-larga")
	players := stack.seedPlayers(t, 2)
	for i, playerID := range players {
		stack.seedStatistics(t, playerID, 7.0+float64(i))
	}

	if err := stack.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el arranque del mercado devolvió %v", err)
	}

	stack.activateScoringStrategy(t, stack.loginToken(t, "admin@nosepudo.test", "superclave-larga"))

	for _, playerID := range players {
		stack.assertPlayerQuote(t, playerID)
	}

	stack.assertRankingCarriesPrice(t, players[0])
}

func (s *stack) activateScoringStrategy(t *testing.T, token string) {
	t.Helper()

	status, body := s.put(t, "/scoring/active-strategy", "Bearer "+token, map[string]any{"id": 1})
	if status != http.StatusOK {
		t.Fatalf("activar la estrategia devolvió %d: %s", status, body)
	}
}

func (s *stack) assertPlayerQuote(t *testing.T, playerID int64) {
	t.Helper()

	expected := s.expectedPrice(t, playerID)
	s.assertPlayerPrice(t, playerID, expected)
	s.assertPlayerQuoteHistory(t, playerID, expected)
	s.assertMarketMakerOffer(t, playerID, expected)
}

func (s *stack) expectedPrice(t *testing.T, playerID int64) model.Credits {
	t.Helper()

	status, body := s.get(t, fmt.Sprintf("/players/%d/score", playerID), "")
	if status != http.StatusOK {
		t.Fatalf("el score del jugador %d devolvió %d: %s", playerID, status, body)
	}

	score := decodeInto[struct {
		Score float64 `json:"score"`
	}](t, body)

	return model.DefaultPricePolicy().Price(score.Score)
}

func (s *stack) assertPlayerPrice(t *testing.T, playerID int64, expected model.Credits) {
	t.Helper()

	status, body := s.get(t, fmt.Sprintf("/players/%d/price", playerID), "")
	if status != http.StatusOK {
		t.Fatalf("el precio del jugador %d devolvió %d: %s", playerID, status, body)
	}

	price := decodeInto[struct {
		Price string `json:"price"`
	}](t, body)
	if price.Price != dto.CreditsADecimal(expected) {
		t.Errorf("el precio del jugador %d es %q, se esperaba %q", playerID, price.Price, dto.CreditsADecimal(expected))
	}
}

func (s *stack) assertPlayerQuoteHistory(t *testing.T, playerID int64, expected model.Credits) {
	t.Helper()

	status, body := s.get(t, fmt.Sprintf("/players/%d/quotes", playerID), "")
	if status != http.StatusOK {
		t.Fatalf("el historial del jugador %d devolvió %d: %s", playerID, status, body)
	}

	quotes := decodeInto[[]struct {
		Price    string `json:"price"`
		Strategy string `json:"strategy"`
	}](t, body)
	if len(quotes) < 2 {
		t.Fatalf("el jugador %d tiene %d cotizaciones, se esperaban la inicial y la del score", playerID, len(quotes))
	}
	if quotes[0].Price != dto.CreditsADecimal(expected) {
		t.Errorf("la cotización vigente del jugador %d es %q, se esperaba %q", playerID, quotes[0].Price, dto.CreditsADecimal(expected))
	}
	if quotes[0].Strategy == model.InitialQuoteStrategy {
		t.Errorf("la cotización vigente del jugador %d sigue siendo la inicial", playerID)
	}
}

func (s *stack) assertMarketMakerOffer(t *testing.T, playerID int64, expected model.Credits) {
	t.Helper()

	var orderPrice int64
	if err := s.db.QueryRowContext(t.Context(),
		`SELECT price FROM market_orders
		 WHERE player_id = $1 AND market_maker AND status IN ('PENDING', 'PARTIALLY_FILLED')`,
		playerID).Scan(&orderPrice); err != nil {
		t.Fatalf("no se pudo leer la oferta del creador para el jugador %d: %v", playerID, err)
	}
	if orderPrice != int64(expected) {
		t.Errorf("la oferta del creador del jugador %d quedó a %d, se esperaba %d", playerID, orderPrice, expected)
	}
}

func (s *stack) assertRankingCarriesPrice(t *testing.T, playerID int64) {
	t.Helper()

	status, body := s.get(t, "/players/ranking", "")
	if status != http.StatusOK {
		t.Fatalf("el ranking devolvió %d: %s", status, body)
	}

	ranking := decodeInto[struct {
		Items []struct {
			Price *string `json:"price"`
			ID    int64   `json:"id"`
		} `json:"items"`
	}](t, body)

	for _, item := range ranking.Items {
		if item.ID == playerID && item.Price == nil {
			t.Errorf("el ranking no trae el precio del jugador %d", playerID)
		}
	}
}
