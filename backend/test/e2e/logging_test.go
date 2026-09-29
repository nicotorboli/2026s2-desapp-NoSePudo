package e2e_test

import (
	"net/http"
	"strings"
	"testing"
)

// Las credenciales de la corrida completa. Son valores reconocibles a propósito:
// la gracia del caso es buscarlos después en lo que el servidor escribió.
const (
	logSubjectEmail = "nico@nosepudo.ar"
	logSubjectPassword = "una-contraseña-muy-reconocible" //nolint:gosec // valor de prueba, no una credencial real
)

// runCompleteFlow ejercita todo lo que esta entrega expone, incluidos los
// caminos que fallan, y devuelve las credenciales que se emitieron para poder
// buscarlas después en los logs.
func runCompleteFlow(t *testing.T, stack *stack) (accessToken, refreshToken string) {
	t.Helper()

	// Alta.
	if status, body := stack.postJSON(t, "/auth/register",
		`{"email":"`+logSubjectEmail+`","password":"`+logSubjectPassword+`"}`); status != http.StatusCreated {
		t.Fatalf("el alta devolvió %d: %v", status, body)
	}

	// Alta duplicada.
	stack.postJSON(t, "/auth/register",
		`{"email":"`+logSubjectEmail+`","password":"`+logSubjectPassword+`"}`)

	// Alta inválida.
	stack.postJSON(t, "/auth/register", `{"email":"sin-arroba","password":"`+logSubjectPassword+`"}`)

	// Inicio de sesión.
	status, session := stack.postJSON(t, "/auth/login",
		`{"email":"`+logSubjectEmail+`","password":"`+logSubjectPassword+`"}`)
	if status != http.StatusOK {
		t.Fatalf("el inicio de sesión devolvió %d: %v", status, session)
	}
	accessToken, refreshToken = tokensOf(t, session)

	// Inicio de sesión con contraseña incorrecta.
	stack.postJSON(t, "/auth/login",
		`{"email":"`+logSubjectEmail+`","password":"la-equivocada"}`)

	// Inicio de sesión de una cuenta que no existe.
	stack.postJSON(t, "/auth/login",
		`{"email":"nadie@nosepudo.ar","password":"`+logSubjectPassword+`"}`)

	// Catálogo con credencial y sin ella.
	stack.get(t, "/players", "Bearer "+accessToken)
	stack.get(t, "/players", "")
	stack.get(t, "/players", "Bearer una-credencial-inventada")

	// Renovación, y después el reuso de la misma credencial.
	_, renewed := stack.postWithBearer(t, "/auth/refresh", refreshToken)
	stack.postWithBearer(t, "/auth/refresh", refreshToken)

	if newRefresh, ok := renewed["refresh_token"].(string); ok && newRefresh != "" {
		refreshToken = newRefresh
	}

	return accessToken, refreshToken
}

// el criterio: un barrido de los eventos de una corrida completa no encuentra
// ninguna contraseña, ninguna credencial, el secreto de firma ni ningún dato
// personal.
func TestCompleteRunLeaksNoSecretAndNoPersonalData(t *testing.T) {
	stack := newStack(t)

	accessToken, refreshToken := runCompleteFlow(t, stack)

	emitted := stack.logs.text()
	if strings.TrimSpace(emitted) == "" {
		t.Fatal("la corrida no emitió ningún evento, así que este caso no probó nada")
	}

	forbidden := map[string]string{
		"la contraseña": logSubjectPassword,
		"la contraseña equivocada": "la-equivocada",
		"la credencial de acceso": accessToken,
		"la credencial de renovación": refreshToken,
		"el secreto de firma": testJWTSecret,
		"el email, que es dato personal": logSubjectEmail,
	}

	for name, secret := range forbidden {
		if strings.Contains(emitted, secret) {
			t.Errorf("los eventos contienen %s", name)
		}
	}

	// El prefijo de un digest de bcrypt tampoco tiene por qué aparecer.
	if strings.Contains(emitted, "$2a$") {
		t.Error("los eventos contienen un hash de contraseña")
	}
}

// el requerimiento: todo rechazo queda registrado con la razón por la que se rechazó.
func TestRefusalsAreLoggedWithTheirReason(t *testing.T) {
	stack := newStack(t)

	runCompleteFlow(t, stack)

	events := stack.logs.lines()

	// Cada tipo de rechazo dejó su evento, y cada uno lleva razón.
	//nolint:gosec // son mensajes de log, no credenciales
	expected := map[string]string{
		"autenticación refutada": "authentication refused",
		"inicio de sesión refutado": "sign-in refused",
		"reuso de credencial de renovación": "refresh credential reused",
	}

	for name, message := range expected {
		found := false
		for _, event := range events {
			if strings.Contains(event, message) {
				found = true
				if !strings.Contains(event, "reason=") {
					t.Errorf("el evento de %s no lleva razón: %s", name, event)
				}
				if !strings.Contains(event, "operation=") {
					t.Errorf("el evento de %s no nombra la operación: %s", name, event)
				}
			}
		}
		if !found {
			t.Errorf("no se registró ningún evento de %s", name)
		}
	}
}

// El cuarto escenario de Escenario:: cuando no se pudo identificar ninguna cuenta, el
// intento se registra igual, con su razón y sin sujeto.
func TestRefusalsWithNoIdentifiableAccountAreStillLogged(t *testing.T) {
	stack := newStack(t)

	// Una petición sin credencial: no hay forma de saber quién la hizo.
	if status, _ := stack.get(t, "/players", ""); status != http.StatusUnauthorized {
		t.Fatalf("el catálogo no rechazó la petición sin credencial")
	}

	events := stack.logs.lines()

	found := false
	for _, event := range events {
		if !strings.Contains(event, "authentication refused") {
			continue
		}
		found = true

		if !strings.Contains(event, "reason=") {
			t.Errorf("el evento no lleva razón: %s", event)
		}
		// Sin credencial verificada no hay sujeto que nombrar, y escribir uno
		// vacío sería peor que no escribir ninguno.
		if strings.Contains(event, "actor=") {
			t.Errorf("el evento nombra un actor que no se pudo identificar: %s", event)
		}
	}

	if !found {
		t.Error("no se registró el rechazo de una petición sin credencial")
	}
}

// Un inicio de sesión fallido no puede registrar la dirección que se intentó,
// porque es dato personal, ni siquiera cuando no existe la cuenta.
func TestRefusedSignInLogsNeitherTheAddressNorThePassword(t *testing.T) {
	stack := newStack(t)

	stack.postJSON(t, "/auth/login",
		`{"email":"nadie@nosepudo.ar","password":"una-contraseña-inventada"}`)

	emitted := stack.logs.text()

	if !strings.Contains(emitted, "sign-in refused") {
		t.Fatal("no se registró el intento de inicio de sesión fallido")
	}
	if strings.Contains(emitted, "nadie@nosepudo.ar") {
		t.Error("el evento registró la dirección que se intentó")
	}
	if strings.Contains(emitted, "una-contraseña-inventada") {
		t.Error("el evento registró la contraseña que se envió")
	}
}

