package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

// accessTokenFor deja una cuenta creada y devuelve su credencial de acceso.
func (s *stack) accessTokenFor(t *testing.T, email, password string) string {
	t.Helper()

	body := s.registerAndLogin(t, email, password)

	token, _ := body["access_token"].(string)
	if token == "" {
		t.Fatalf("el login no devolvió credencial: %v", body)
	}

	return token
}

// US3, escenario 5: con credencial válida el catálogo se sirve.
func TestCatalogIsServedWithAValidCredential(t *testing.T) {
	stack := newStack(t)
	token := stack.accessTokenFor(t, "nico@nosepudo.ar", "una-contraseña")

	status, body := stack.get(t, "/players", "Bearer "+token)

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", status, body)
	}

	var players []map[string]any
	if err := json.Unmarshal([]byte(body), &players); err != nil {
		t.Fatalf("la respuesta no es una lista de jugadores: %v (%s)", err, body)
	}
	if len(players) == 0 {
		t.Error("el catálogo vino vacío, se esperaban los jugadores sembrados por init.sql")
	}
}

// US3, escenarios 1, 3 y 4, y SC-001: sin credencial, con una alterada, o con
// una presentada de forma inesperada, el catálogo se niega.
func TestCatalogIsRefusedWithoutAUsableCredential(t *testing.T) {
	stack := newStack(t)
	token := stack.accessTokenFor(t, "nico@nosepudo.ar", "una-contraseña")

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("la credencial no es un JWS compacto")
	}

	cases := []struct {
		name          string
		authorization string
	}{
		{"sin credencial", ""},
		{"sin esquema", token},
		{"valor vacío", "Bearer"},
		{"esquema desconocido", "Basic " + token},
		{"payload alterado", "Bearer " + parts[0] + "." + parts[1][:len(parts[1])-2] + "XY." + parts[2]},
		{"firma alterada", "Bearer " + parts[0] + "." + parts[1] + "." + parts[2][:len(parts[2])-2] + "XY"},
		{"sin firma", "Bearer " + parts[0] + "." + parts[1] + "."},
		{"no es una credencial", "Bearer esto-no-es-un-jwt"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body := stack.get(t, "/players", c.authorization)

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401: %s", status, body)
			}

			// El cuerpo es el error de siempre y no filtra por qué falló.
			var decoded map[string]string
			if err := json.Unmarshal([]byte(body), &decoded); err != nil {
				t.Fatalf("la respuesta no es JSON: %v (%s)", err, body)
			}
			if decoded["error"] != "authentication required" {
				t.Errorf("error = %q, se esperaba el mensaje del contrato", decoded["error"])
			}
			if len(decoded) != 1 {
				t.Errorf("la respuesta tiene %d campos, el contrato fija 1: %v", len(decoded), decoded)
			}
		})
	}
}

// US3, escenario 2: una credencial expirada se niega. Se consigue de verdad
// levantando el stack con un tiempo de vida de un instante.
func TestCatalogIsRefusedWithAnExpiredCredential(t *testing.T) {
	stack := newStackWithAccessTTL(t, time.Second)
	token := stack.accessTokenFor(t, "nico@nosepudo.ar", "una-contraseña")

	// Antes de expirar sirve.
	if status, body := stack.get(t, "/players", "Bearer "+token); status != http.StatusOK {
		t.Fatalf("la credencial recién emitida ya no servía: %d %s", status, body)
	}

	time.Sleep(2 * time.Second)

	status, body := stack.get(t, "/players", "Bearer "+token)
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401 con la credencial expirada: %s", status, body)
	}
}

// SC-001: el estado del sistema es idéntico antes y después de cada intento
// rechazado, y FR-008 exige que se rechace antes de tocar la persistencia.
func TestRefusedRequestsLeaveNoTrace(t *testing.T) {
	stack := newStack(t)
	stack.accessTokenFor(t, "nico@nosepudo.ar", "una-contraseña")

	before := stack.countUsers(t)

	for range 5 {
		if status, _ := stack.get(t, "/players", ""); status != http.StatusUnauthorized {
			t.Fatalf("status = %d, se esperaba 401", status)
		}
	}

	if after := stack.countUsers(t); after != before {
		t.Errorf("el estado cambió: había %d cuentas y quedaron %d", before, after)
	}
}

// El caso borde de la spec: una credencial válida junto a un cuerpo malformado.
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
