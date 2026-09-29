package e2e_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// claimsOf lee el payload del JWT sin verificar la firma, que es exactamente
// lo que puede hacer cualquiera que tenga la credencial en la mano. Sirve para
// comprobar qué dice y, sobre todo, qué no dice.
func claimsOf(t *testing.T, token string) map[string]any {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("la credencial no es un JWS compacto: %q", token)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("no se pudo decodificar el payload: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("el payload no es JSON: %v", err)
	}

	return claims
}

// registerAndLogin deja una cuenta creada y devuelve el cuerpo del login.
func (s *stack) registerAndLogin(t *testing.T, email, password string) map[string]any {
	t.Helper()

	body := `{"email":"` + email + `","password":"` + password + `"}`
	if status, response := s.postJSON(t, "/auth/register", body); status != http.StatusCreated {
		t.Fatalf("el alta devolvió %d: %v", status, response)
	}

	status, response := s.postJSON(t, "/auth/login", body)
	if status != http.StatusOK {
		t.Fatalf("el login devolvió %d: %v", status, response)
	}

	return response
}

// Escenario: credenciales correctas devuelven la credencial y cuándo expira.
func TestLoginReturnsASessionCredential(t *testing.T) {
	stack := newStack(t)

	body := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	accessToken, _ := body["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("no vino credencial de acceso: %v", body)
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v", body["token_type"])
	}

	expiresAt, err := time.Parse(time.RFC3339, body["access_expires_at"].(string))
	if err != nil {
		t.Fatalf("access_expires_at no es RFC 3339: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("la credencial ya venía expirada: %v", expiresAt)
	}

	// Escenario: lleva la identidad de la cuenta, su privilegio y un vencimiento.
	claims := claimsOf(t, accessToken)
	if claims["sub"] == "" || claims["sub"] == nil {
		t.Error("la credencial no nombra la cuenta")
	}
	if claims["priv"] != "user" {
		t.Errorf("priv = %v, se esperaba user", claims["priv"])
	}
	if claims["typ"] != "access" {
		t.Errorf("typ = %v, se esperaba access", claims["typ"])
	}
	if claims["exp"] == nil {
		t.Error("la credencial no declara cuándo expira")
	}
	if claims["sid"] == nil || claims["sid"] == "" {
		t.Error("la credencial no nombra su familia de sesión")
	}
}

// Escenario:escenarios 2 y 3: una contraseña incorrecta y una cuenta inexistente
// responden exactamente lo mismo, y ninguna emite credencial.
func TestLoginAnswersIdenticallyForBothFailures(t *testing.T) {
	stack := newStack(t)
	stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	wrongPasswordStatus, wrongPasswordBody := stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"la-equivocada"}`)
	unknownAccountStatus, unknownAccountBody := stack.postJSON(t, "/auth/login",
		`{"email":"nadie@nosepudo.ar","password":"la-equivocada"}`)

	if wrongPasswordStatus != http.StatusUnauthorized {
		t.Errorf("contraseña incorrecta devolvió %d, se esperaba 401", wrongPasswordStatus)
	}
	if unknownAccountStatus != http.StatusUnauthorized {
		t.Errorf("cuenta inexistente devolvió %d, se esperaba 401", unknownAccountStatus)
	}
	if !reflect.DeepEqual(wrongPasswordBody, unknownAccountBody) {
		t.Errorf("las respuestas difieren y revelan si la cuenta existe: %v vs %v",
			wrongPasswordBody, unknownAccountBody)
	}
	if _, issued := wrongPasswordBody["access_token"]; issued {
		t.Error("se emitió una credencial pese al fallo")
	}
}

// Escenario: una entrada inválida se rechaza sin buscar ni verificar nada.
func TestLoginRejectsInvalidInput(t *testing.T) {
	stack := newStack(t)

	cases := []struct {
		name string
		body string
	}{
		{"cuerpo vacío", ``},
		{"email ausente", `{"password":"una-contraseña"}`},
		{"email en blanco", `{"email":"","password":"una-contraseña"}`},
		{"email con forma imposible", `{"email":"sin-arroba","password":"una-contraseña"}`},
		{"contraseña ausente", `{"email":"nico@nosepudo.ar"}`},
		{"campo desconocido", `{"email":"nico@nosepudo.ar","password":"x","remember":true}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body := stack.postJSON(t, "/auth/login", c.body)

			if status != http.StatusBadRequest {
				t.Errorf("status = %d, se esperaba 400: %v", status, body)
			}
		})
	}
}

// Escenario: cualquiera que tenga la credencial puede leer su
// contenido, así que ahí no puede haber nada que duela divulgar.
func TestLoginCredentialCarriesNothingDamaging(t *testing.T) {
	const password = "una-contraseña-reconocible" //nolint:gosec // valor de prueba, no una credencial real
	stack := newStack(t)

	body := stack.registerAndLogin(t, "nico@nosepudo.ar", password)
	accessToken := body["access_token"].(string)

	claims := claimsOf(t, accessToken)
	encoded, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("no se pudieron serializar los claims: %v", err)
	}

	readable := strings.ToLower(string(encoded))
	for _, forbidden := range []string{password, "password", "hash", "$2a$", "nico@nosepudo.ar"} {
		if strings.Contains(readable, strings.ToLower(forbidden)) {
			t.Errorf("la credencial contiene %q en claro: %s", forbidden, encoded)
		}
	}
}

// El caso borde, del lado del login: otra forma de la misma
// dirección resuelve a la misma cuenta.
func TestLoginAcceptsTheIdentifierInAnotherCase(t *testing.T) {
	stack := newStack(t)
	stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	status, body := stack.postJSON(t, "/auth/login",
		`{"email":" NICO@NoSePudo.AR ","password":"una-contraseña"}`)

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %v", status, body)
	}
}

// El caso borde: dos inicios de sesión de la misma cuenta funcionan
// de forma independiente y ninguno invalida al otro.
func TestTwoSignInsAreIndependent(t *testing.T) {
	stack := newStack(t)

	first := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	status, second := stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusOK {
		t.Fatalf("el segundo inicio de sesión devolvió %d", status)
	}

	firstToken := first["access_token"].(string)
	secondToken := second["access_token"].(string)

	if firstToken == secondToken {
		t.Error("los dos inicios de sesión devolvieron la misma credencial")
	}

	firstClaims := claimsOf(t, firstToken)
	secondClaims := claimsOf(t, secondToken)

	if firstClaims["jti"] == secondClaims["jti"] {
		t.Error("las dos credenciales comparten identificador")
	}
	if firstClaims["sid"] == secondClaims["sid"] {
		t.Error("los dos inicios de sesión comparten familia: cerrar uno cerraría el otro")
	}
	if firstClaims["sub"] != secondClaims["sub"] {
		t.Error("las credenciales nombran cuentas distintas")
	}
}

