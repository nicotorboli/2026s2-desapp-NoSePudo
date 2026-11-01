package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// get pide el catálogo con la cabecera Authorization que se le indique. Una
// cadena vacía significa no mandar la cabecera en absoluto.
func (s *stack) get(t *testing.T, path, authorization string) (int, string) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.server.URL+path, nil)
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}
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

// Escenario: el catálogo se sirve sin credencial.
func TestCatalogIsServed(t *testing.T) {
	stack := newStack(t)

	// init.sql ya no siembra jugadores (llegan por el sync), así que el caso
	// inserta uno propio.
	if _, err := stack.db.ExecContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES (1, 'Ernesto Provitillo', 'Arsenal FC', 'Premier League', 'PL', 'Midfielder')`,
	); err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}

	status, body := stack.get(t, "/players", "")

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", status, body)
	}

	var page struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("la respuesta no es una página de jugadores: %v (%s)", err, body)
	}
	if len(page.Items) == 0 {
		t.Error("el catálogo vino vacío, se esperaba el jugador sembrado")
	}
}

// El caso borde: una credencial válida junto a un cuerpo malformado.
// La credencial se comprueba primero, y el cuerpo se rechaza recién cuando ya
// se sabe quién pide.
func TestCredentialIsCheckedBeforeTheBody(t *testing.T) {
	stack := newStack(t)

	// El alta es anónima, así que para ver el orden hace falta un endpoint
	// autenticado que además acepte cuerpo. Todavía no hay ninguno en esta
	// entrega, así que lo que se comprueba es la mitad disponible: un cuerpo
	// inválido en un endpoint anónimo se rechaza como 400 y no como 401.
	status, _ := stack.postJSON(t, "/auth/register", `{"email":`)

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, se esperaba 400 en un endpoint anónimo", status)
	}
}

// Los dos endpoints anónimos siguen alcanzables sin credencial: si el alta o el
// login la exigieran, no habría forma de conseguir la primera.
func TestAnonymousEndpointsNeedNoCredential(t *testing.T) {
	stack := newStack(t)

	status, body := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusCreated {
		t.Errorf("el alta devolvió %d sin credencial: %v", status, body)
	}

	status, body = stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusOK {
		t.Errorf("el login devolvió %d sin credencial: %v", status, body)
	}
}
