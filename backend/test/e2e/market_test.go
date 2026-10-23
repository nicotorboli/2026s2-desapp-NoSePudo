package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func (s *stack) post(t *testing.T, path, authorization string, payload any) (int, string) {
	t.Helper()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("no se pudo serializar el cuerpo: %v", err)
	}

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.server.URL+path, bytes.NewReader(encoded))
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

func decodeInto[T any](t *testing.T, body string) T {
	t.Helper()

	var decoded T
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("no se pudo leer la respuesta %q: %v", body, err)
	}

	return decoded
}

type portfolioBody struct {
	AvailableCash string `json:"availableCash"`
	TotalValue    string `json:"totalValue"`
	Positions     []struct {
		PlayerName       string `json:"playerName"`
		CurrentPrice     string `json:"currentPrice"`
		CurrentValuation string `json:"currentValuation"`
		UnrealizedResult string `json:"unrealizedResult"`
		PlayerID         int64  `json:"playerId"`
		Quantity         int64  `json:"quantity"`
	} `json:"positions"`
	UserID int64 `json:"userId"`
}

func (s *stack) comprarCadaJugador(t *testing.T, token string, quien int, players []int64, cantidad int64) {
	t.Helper()

	for _, playerID := range players {
		status, body := s.post(t, "/orders/buy", "Bearer "+token, map[string]any{
			"playerId": playerID,
			"quantity": cantidad,
		})
		if status != http.StatusCreated {
			t.Fatalf("la compra del usuario %d sobre el jugador %d devolvió %d: %s", quien, playerID, status, body)
		}

		order := decodeInto[struct {
			Status string `json:"status"`
		}](t, body)
		if order.Status != string(model.StatusFilled) {
			t.Errorf("la compra del usuario %d sobre el jugador %d quedó en %s, se esperaba FILLED", quien, playerID, order.Status)
		}
	}
}

func (s *stack) verificarPatrimonio(t *testing.T, token string, quien int, userID int64, esperadas int, cantidad int64) {
	t.Helper()

	status, body := s.get(t, fmt.Sprintf("/users/%d/portfolio", userID), "Bearer "+token)
	if status != http.StatusOK {
		t.Fatalf("el patrimonio del usuario %d devolvió %d: %s", quien, status, body)
	}

	portfolio := decodeInto[portfolioBody](t, body)
	if len(portfolio.Positions) != esperadas {
		t.Fatalf("el usuario %d informa %d posiciones, se esperaban %d", quien, len(portfolio.Positions), esperadas)
	}

	for _, position := range portfolio.Positions {
		if position.Quantity != cantidad {
			t.Errorf("el usuario %d tiene %d tokens del jugador %d, se esperaban %d",
				quien, position.Quantity, position.PlayerID, cantidad)
		}
		if position.CurrentValuation == "" || position.CurrentPrice == "" {
			t.Errorf("el usuario %d tiene una posición sin valuar en el jugador %d", quien, position.PlayerID)
		}
		if position.PlayerName == "" {
			t.Errorf("el usuario %d tiene una posición sin nombre de jugador", quien)
		}
	}
}

func TestElEscenarioDeEvaluacionCuatroUsuariosCincoJugadoresYLasCotizacionesSeMueven(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, "admin@nosepudo.test", "superclave-larga")
	players := stack.seedPlayers(t, 5)

	if err := stack.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el arranque del mercado devolvió %v", err)
	}

	const (
		usuarios  = 4
		comprados = 4
	)

	correos := make([]string, 0, usuarios)
	tokens := make([]string, 0, usuarios)
	for i := range usuarios {
		correo := fmt.Sprintf("usuario%d@nosepudo.test", i+1)
		correos = append(correos, correo)
		tokens = append(tokens, stack.accessTokenFor(t, correo, "una-contraseña-larga"))
	}

	for i, token := range tokens {
		stack.comprarCadaJugador(t, token, i+1, players, comprados)
	}

	nuevosPrecios := map[int64]model.Credits{
		players[0]: model.CreditsFromUnits(3),
		players[1]: model.CreditsFromUnits(2),
		players[2]: model.Credits(model.CreditScale / 2),
		players[3]: model.CreditsFromUnits(5),
		players[4]: model.CreditsFromUnits(1),
	}

	for playerID, price := range nuevosPrecios {
		if err := stack.marketMaker.RecordQuote(t.Context(), playerID, price, "rendimiento", "1"); err != nil {
			t.Fatalf("recotizar al jugador %d devolvió %v", playerID, err)
		}
	}

	for i, token := range tokens {
		stack.verificarPatrimonio(t, token, i+1, stack.userIDOf(t, correos[i]), len(players), comprados)
	}

	for _, playerID := range players {
		if total := stack.tokensOfPlayer(t, playerID); total != 100 {
			t.Errorf("el jugador %d terminó con %d tokens, se esperaban exactamente 100", playerID, total)
		}
	}

	if unbalanced := stack.unbalancedSettlements(t); unbalanced != 0 {
		t.Errorf("hay %d liquidaciones que no suman cero", unbalanced)
	}

	if registradas := stack.countRows(t, `SELECT COUNT(*) FROM market_trades`); registradas != usuarios*len(players) {
		t.Errorf("se registraron %d operaciones, se esperaban %d", registradas, usuarios*len(players))
	}
}

func TestElPatrimonioAjenoSeNiegaYElSuperusuarioLoVe(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, "admin@nosepudo.test", "superclave-larga")

	propio := stack.accessTokenFor(t, "propio@nosepudo.test", "una-contraseña-larga")
	stack.accessTokenFor(t, "ajeno@nosepudo.test", "una-contraseña-larga")

	ajenoID := stack.userIDOf(t, "ajeno@nosepudo.test")

	status, body := stack.get(t, fmt.Sprintf("/users/%d/portfolio", ajenoID), "Bearer "+propio)
	if status != http.StatusForbidden {
		t.Errorf("leer el patrimonio ajeno devolvió %d, se esperaba 403: %s", status, body)
	}

	status, body = stack.get(t, fmt.Sprintf("/users/%d/transactions", ajenoID), "Bearer "+propio)
	if status != http.StatusForbidden {
		t.Errorf("leer el historial ajeno devolvió %d, se esperaba 403: %s", status, body)
	}

	admin := stack.loginToken(t, "admin@nosepudo.test", "superclave-larga")

	status, body = stack.get(t, fmt.Sprintf("/users/%d/portfolio", ajenoID), "Bearer "+admin)
	if status != http.StatusOK {
		t.Errorf("el superusuario no pudo leer el patrimonio ajeno: %d %s", status, body)
	}
}

func TestLasRutasDelMercadoExigenCredencial(t *testing.T) {
	stack := newStack(t)

	rutas := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/orders"},
		{http.MethodGet, "/users/1/portfolio"},
		{http.MethodGet, "/users/1/transactions"},
	}

	for _, ruta := range rutas {
		t.Run(ruta.method+" "+ruta.path, func(t *testing.T) {
			status, _ := stack.get(t, ruta.path, "")
			if status != http.StatusUnauthorized {
				t.Errorf("sin credencial devolvió %d, se esperaba 401", status)
			}
		})
	}

	status, _ := stack.post(t, "/orders/buy", "", map[string]any{"playerId": 1, "quantity": 1})
	if status != http.StatusUnauthorized {
		t.Errorf("POST /orders/buy sin credencial devolvió %d, se esperaba 401", status)
	}
}

func (s *stack) userIDOf(t *testing.T, email string) int64 {
	t.Helper()

	var id int64
	if err := s.db.QueryRowContext(t.Context(), `SELECT id FROM users WHERE email = $1`, email).Scan(&id); err != nil {
		t.Fatalf("no se pudo resolver el identificador de %q: %v", email, err)
	}

	return id
}

func (s *stack) loginToken(t *testing.T, email, password string) string {
	t.Helper()

	status, body := s.post(t, "/auth/login", "", map[string]any{"email": email, "password": password})
	if status != http.StatusOK {
		t.Fatalf("el login de %q devolvió %d: %s", email, status, body)
	}

	session := decodeInto[struct {
		AccessToken string `json:"access_token"`
	}](t, body)
	if session.AccessToken == "" {
		t.Fatalf("el login de %q no devolvió credencial: %s", email, body)
	}

	return session.AccessToken
}
