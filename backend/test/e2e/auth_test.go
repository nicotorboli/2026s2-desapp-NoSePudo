package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// postJSON hace la petición como la haría un cliente y devuelve el status y el
// cuerpo decodificado.
func (s *stack) postJSON(t *testing.T, path, body string) (int, map[string]any) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.server.Client().Do(request)
	if err != nil {
		t.Fatalf("la petición falló: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("la respuesta no es JSON: %v", err)
	}

	return response.StatusCode, decoded
}

// US1, escenario 1: con datos válidos la cuenta existe y es de usuario común.
func TestRegisterCreatesACommonUserAccount(t *testing.T) {
	stack := newStack(t)

	status, body := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201: %v", status, body)
	}

	var (
		email     string
		privilege int16
		active    bool
	)
	err := stack.db.QueryRowContext(t.Context(),
		"SELECT email, privilege, active FROM users WHERE id = $1", int64(body["id"].(float64)),
	).Scan(&email, &privilege, &active)
	if err != nil {
		t.Fatalf("la cuenta no quedó guardada: %v", err)
	}

	if email != "nico@nosepudo.ar" {
		t.Errorf("email guardado = %q", email)
	}
	// Se compara ensanchando el nivel a int16 y no truncando la columna a
	// uint8: truncar haría que un valor corrupto de 258 pasara por superusuario.
	if privilege != int16(model.PrivilegeUser) {
		t.Errorf("privilege = %d, se esperaba usuario común", privilege)
	}
	if !active {
		t.Error("la cuenta quedó inactiva")
	}
}

// US1, escenario 2: un identificador ya en uso se rechaza y la cuenta
// existente queda intacta.
func TestRegisterRefusesADuplicateIdentifier(t *testing.T) {
	stack := newStack(t)

	status, first := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusCreated {
		t.Fatalf("el primer alta devolvió %d", status)
	}

	status, body := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"otra-contraseña"}`)

	if status != http.StatusConflict {
		t.Fatalf("status = %d, se esperaba 409: %v", status, body)
	}
	if stack.countUsers(t) != 1 {
		t.Errorf("hay %d cuentas, se esperaba 1", stack.countUsers(t))
	}

	// La cuenta existente conserva su hash original.
	var hash string
	if err := stack.db.QueryRowContext(t.Context(),
		"SELECT password_hash FROM users WHERE id = $1", int64(first["id"].(float64)),
	).Scan(&hash); err != nil {
		t.Fatalf("no se pudo leer la cuenta existente: %v", err)
	}
	if hash == "" {
		t.Error("la cuenta existente fue modificada")
	}
}

// El caso borde de la spec: mayúsculas y espacios resuelven a la misma cuenta.
func TestRegisterNormalizesTheIdentifier(t *testing.T) {
	stack := newStack(t)

	if status, _ := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`); status != http.StatusCreated {
		t.Fatalf("el primer alta devolvió %d", status)
	}

	status, _ := stack.postJSON(t, "/auth/register",
		`{"email":"  NICO@NoSePudo.AR  ","password":"otra-contraseña"}`)

	if status != http.StatusConflict {
		t.Errorf("status = %d, se esperaba 409: una dirección se volvió dos cuentas", status)
	}
	if count := stack.countUsers(t); count != 1 {
		t.Errorf("hay %d cuentas, se esperaba 1", count)
	}
}

// US1, escenario 3 y SC-003: toda entrada inválida se rechaza sin crear nada.
func TestRegisterCreatesNothingForInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"email ausente", `{"password":"una-contraseña"}`},
		{"email sin arroba", `{"email":"sin-arroba","password":"una-contraseña"}`},
		{"contraseña ausente", `{"email":"nico@nosepudo.ar"}`},
		{"contraseña corta", `{"email":"nico@nosepudo.ar","password":"1234567"}`},
		{"contraseña de 73 bytes", `{"email":"nico@nosepudo.ar","password":"` + strings.Repeat("a", 73) + `"}`},
		{"campo desconocido", `{"email":"nico@nosepudo.ar","password":"una-contraseña","admin":true}`},
		{"JSON malformado", `{"email":`},
		{"cuerpo vacío", ``},
	}

	stack := newStack(t)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := stack.countUsers(t)

			status, body := stack.postJSON(t, "/auth/register", c.body)

			if status != http.StatusBadRequest {
				t.Errorf("status = %d, se esperaba 400: %v", status, body)
			}
			if after := stack.countUsers(t); after != before {
				t.Errorf("se crearon %d cuentas con una entrada inválida", after-before)
			}
		})
	}
}

// US1, escenario 4: una petición que trae un nivel de privilegio se rechaza
// como campo inesperado, y bajo ninguna circunstancia se crea una cuenta con
// el privilegio que vino en el cuerpo.
func TestRegisterRefusesARequestCarryingAPrivilege(t *testing.T) {
	stack := newStack(t)

	status, body := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña","privilege":"superuser"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400: %v", status, body)
	}
	if message, _ := body["error"].(string); !strings.Contains(message, "privilege") {
		t.Errorf("error = %v, se esperaba que nombrara el campo sobrante", body["error"])
	}
	if count := stack.countUsers(t); count != 0 {
		t.Fatalf("se creó una cuenta: hay %d", count)
	}

	// Y por las dudas: no existe ninguna cuenta con privilegio de superusuario.
	var superusers int
	if err := stack.db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM users WHERE privilege = $1", int16(model.PrivilegeSuperuser),
	).Scan(&superusers); err != nil {
		t.Fatalf("no se pudieron contar los superusuarios: %v", err)
	}
	if superusers != 0 {
		t.Errorf("hay %d superusuarios: un cliente se otorgó privilegio", superusers)
	}
}

// US1, escenario 5: del registro guardado no se puede recuperar la contraseña.
func TestRegisterStoresNoRecoverablePassword(t *testing.T) {
	const password = "una-contraseña-reconocible" //nolint:gosec // valor de prueba, no una credencial real
	stack := newStack(t)

	status, body := stack.postJSON(t, "/auth/register",
		`{"email":"nico@nosepudo.ar","password":"`+password+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201", status)
	}

	// La respuesta no lleva la contraseña en ninguna forma.
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}
	if strings.Contains(string(encoded), password) {
		t.Errorf("la respuesta contiene la contraseña: %s", encoded)
	}

	// La fila tampoco.
	var hash string
	if err := stack.db.QueryRowContext(t.Context(),
		"SELECT password_hash FROM users WHERE email = $1", "nico@nosepudo.ar",
	).Scan(&hash); err != nil {
		t.Fatalf("no se pudo leer la cuenta: %v", err)
	}

	if strings.Contains(hash, password) {
		t.Error("la fila contiene la contraseña en claro")
	}
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("el hash guardado no parece un digest de bcrypt: %q", hash)
	}
	if len(hash) != 60 {
		t.Errorf("el hash mide %d caracteres, un digest de bcrypt mide 60", len(hash))
	}
}
