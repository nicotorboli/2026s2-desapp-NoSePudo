package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// fakeAccessToken es el valor que el service simulado devuelve como
// credencial. Es una cadena de prueba, no una credencial real.
const fakeAccessToken = "una-credencial" //nolint:gosec // valor de prueba, no una credencial real

// mockAuthService devuelve lo que el caso necesite. Lo que se prueba acá es el
// borde HTTP: los status, la forma del cuerpo y qué llega al service.
type mockAuthService struct {
	gotEmail        string
	gotPassword     string
	gotCredentialID string
	gotSessionID    string
	err             error
	session         model.Session
	user            model.User
	gotUserID       int64
	registerCalled  int
	loginCalled     int
	refreshCalled   int
	logoutCalled    int
}

func (m *mockAuthService) Register(_ context.Context, email, password string) (model.User, error) {
	m.registerCalled++
	m.gotEmail = email
	m.gotPassword = password
	return m.user, m.err
}

func (m *mockAuthService) Login(_ context.Context, email, password string) (model.Session, error) {
	m.loginCalled++
	m.gotEmail = email
	m.gotPassword = password
	return m.session, m.err
}

func (m *mockAuthService) Refresh(
	_ context.Context,
	presentedID string,
	userID int64,
	sessionID string,
) (model.Session, error) {
	m.refreshCalled++
	m.gotCredentialID = presentedID
	m.gotUserID = userID
	m.gotSessionID = sessionID
	return m.session, m.err
}

func (m *mockAuthService) Logout(_ context.Context, userID int64, sessionID string) error {
	m.logoutCalled++
	m.gotUserID = userID
	m.gotSessionID = sessionID
	return m.err
}

// callRegister corre el endpoint a través de Wrap, que es como lo ve un
// cliente: el error que devuelve el controller ya llega convertido en
// respuesta.
func callRegister(t *testing.T, service *mockAuthService, body string) (int, map[string]any) {
	t.Helper()
	return call(t, controller.NewAuthController(service).Register(), "/auth/register", body)
}

func callLogin(t *testing.T, service *mockAuthService, body string) (int, map[string]any) {
	t.Helper()
	return call(t, controller.NewAuthController(service).Login(), "/auth/login", body)
}

func call(t *testing.T, endpoint httphandler.Endpoint, path, body string) (int, map[string]any) {
	t.Helper()

	handler := httphandler.Wrap(endpoint, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	handler.ServeHTTP(recorder, request)

	var decoded map[string]any
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("la respuesta no es JSON: %v (%s)", err, recorder.Body.String())
		}
	}

	return recorder.Code, decoded
}

func TestRegisterEndpointReturns201WithTheAccountResponse(t *testing.T) {
	service := &mockAuthService{
		user: model.User{ID: 42, Email: "nico@nosepudo.ar", Privilege: model.PrivilegeUser, Active: true},
	}

	status, body := callRegister(t, service, `{"email":" Nico@NoSePudo.AR ","password":"una-contraseña"}`)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201: %v", status, body)
	}

	// El contrato fija id y email, y nada más.
	if got, ok := body["id"].(float64); !ok || int64(got) != 42 {
		t.Errorf("id = %v, se esperaba 42", body["id"])
	}
	if body["email"] != "nico@nosepudo.ar" {
		t.Errorf("email = %v", body["email"])
	}
	if len(body) != 2 {
		t.Errorf("la respuesta tiene %d campos, el contrato fija 2: %v", len(body), body)
	}

	// El controller pasa lo que recibió; normalizar es del dominio.
	if service.gotEmail != " Nico@NoSePudo.AR " {
		t.Errorf("el service recibió %q", service.gotEmail)
	}
	if service.gotPassword != "una-contraseña" {
		t.Errorf("el service recibió otra contraseña")
	}
}

func TestRegisterEndpointReturns409WhenTheEmailIsTaken(t *testing.T) {
	// El service decora el error al subirlo; el centinela sigue en la cadena.
	service := &mockAuthService{err: fmt.Errorf("crear la cuenta: %w", model.ErrEmailTaken)}

	status, body := callRegister(t, service, `{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)

	if status != http.StatusConflict {
		t.Fatalf("status = %d, se esperaba 409", status)
	}
	if body["error"] != "email already registered" {
		t.Errorf("error = %v, se esperaba el mensaje del contrato", body["error"])
	}
}

// el requerimiento y el cuarto escenario de Escenario:: un privilege en el cuerpo se rechaza,
// y el service ni se entera.
func TestRegisterEndpointRefusesAPrivilegeField(t *testing.T) {
	service := &mockAuthService{}

	status, body := callRegister(t, service,
		`{"email":"nico@nosepudo.ar","password":"una-contraseña","privilege":"superuser"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", status)
	}
	if message, _ := body["error"].(string); !strings.Contains(message, "privilege") {
		t.Errorf("error = %v, se esperaba que nombrara el campo sobrante", body["error"])
	}
	if service.registerCalled != 0 {
		t.Error("el service corrió pese a que el cuerpo era inválido")
	}
}

// el requerimiento: una entrada inválida se rechaza sin ejecutar lógica de negocio.
func TestRegisterEndpointReturns400ForInvalidBodies(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"cuerpo vacío", ``},
		{"JSON malformado", `{"email":`},
		{"email ausente", `{"password":"una-contraseña"}`},
		{"email en blanco", `{"email":" ","password":"una-contraseña"}`},
		{"email sin arroba", `{"email":"sin-arroba","password":"una-contraseña"}`},
		{"contraseña ausente", `{"email":"nico@nosepudo.ar"}`},
		{"contraseña de 7 bytes", `{"email":"nico@nosepudo.ar","password":"1234567"}`},
		{"contraseña de 73 bytes", `{"email":"nico@nosepudo.ar","password":"` + strings.Repeat("a", 73) + `"}`},
		{"campo desconocido", `{"email":"nico@nosepudo.ar","password":"una-contraseña","admin":true}`},
		{"dos objetos JSON", `{"email":"a@b.c","password":"una-contraseña"}{"email":"d@e.f"}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service := &mockAuthService{}

			status, body := callRegister(t, service, c.body)

			if status != http.StatusBadRequest {
				t.Errorf("status = %d, se esperaba 400: %v", status, body)
			}
			if service.registerCalled != 0 {
				t.Error("el service corrió pese a que el cuerpo era inválido")
			}
			if _, hasError := body["error"]; !hasError {
				t.Errorf("la respuesta no tiene el campo error que fija el contrato: %v", body)
			}
		})
	}
}

// el requerimiento: el mensaje de una validación fallida no repite la contraseña.
func TestRegisterEndpointNeverEchoesTheSubmittedPassword(t *testing.T) {
	const secret = "contraseña-secretisima" //nolint:gosec // valor de prueba, no una credencial real
	service := &mockAuthService{}

	_, body := callRegister(t, service, `{"email":"sin-arroba","password":"`+secret+`"}`)

	if message, _ := body["error"].(string); strings.Contains(message, secret) {
		t.Errorf("la respuesta repite la contraseña: %q", message)
	}
}

// Un fallo que no es de dominio no revela su causa.
func TestRegisterEndpointReturns500ForAnUnexpectedFailure(t *testing.T) {
	service := &mockAuthService{err: errors.New("la base se cayó")}

	status, body := callRegister(t, service, `{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)

	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, se esperaba 500", status)
	}
	if message, _ := body["error"].(string); strings.Contains(message, "la base se cayó") {
		t.Errorf("la respuesta filtra la causa interna: %q", message)
	}
}

func TestLoginEndpointReturns200WithTheSessionResponse(t *testing.T) {
	expiry := time.Date(2026, time.September, 28, 12, 15, 0, 0, time.UTC)
	service := &mockAuthService{
		session: model.Session{AccessExpiresAt: expiry, AccessToken: fakeAccessToken},
	}

	status, body := callLogin(t, service, `{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %v", status, body)
	}
	if body["access_token"] != fakeAccessToken {
		t.Errorf("access_token = %v", body["access_token"])
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v, el contrato lo fija en Bearer", body["token_type"])
	}
	if body["access_expires_at"] != expiry.Format(time.RFC3339) {
		t.Errorf("access_expires_at = %v, se esperaba RFC 3339", body["access_expires_at"])
	}
}

// el requerimiento: los dos modos de fallar responden idéntico, byte por byte.
func TestLoginEndpointAnswersIdenticallyForBothFailureCauses(t *testing.T) {
	wrongPassword := &mockAuthService{err: fmt.Errorf("comparar: %w", model.ErrInvalidCredentials)}
	unknownAccount := &mockAuthService{err: fmt.Errorf("buscar: %w", model.ErrInvalidCredentials)}

	firstStatus, firstBody := callLogin(t, wrongPassword, `{"email":"nico@nosepudo.ar","password":"mal"}`)
	secondStatus, secondBody := callLogin(t, unknownAccount, `{"email":"nadie@nosepudo.ar","password":"mal"}`)

	if firstStatus != http.StatusUnauthorized || secondStatus != http.StatusUnauthorized {
		t.Fatalf("status = %d y %d, se esperaba 401 en ambos", firstStatus, secondStatus)
	}
	if firstBody["error"] != "invalid credentials" {
		t.Errorf("error = %v, se esperaba el mensaje del contrato", firstBody["error"])
	}
	if !reflect.DeepEqual(firstBody, secondBody) {
		t.Errorf("las respuestas difieren y revelan si la cuenta existe: %v vs %v", firstBody, secondBody)
	}
}

// el requerimiento: la respuesta no lleva nada que sea dañino divulgar.
func TestLoginEndpointResponseCarriesNoSecret(t *testing.T) {
	service := &mockAuthService{
		session: model.Session{AccessToken: fakeAccessToken, AccessExpiresAt: time.Now()},
	}

	_, body := callLogin(t, service, `{"email":"nico@nosepudo.ar","password":"la-contraseña"}`)

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("no se pudo serializar: %v", err)
	}
	for _, forbidden := range []string{"la-contraseña", "password", "hash"} {
		if strings.Contains(strings.ToLower(string(encoded)), strings.ToLower(forbidden)) {
			t.Errorf("la respuesta contiene %q: %s", forbidden, encoded)
		}
	}
}

// El cuarto escenario de Escenario:: una entrada inválida se rechaza sin buscar ni
// verificar credencial alguna.
func TestLoginEndpointReturns400WithoutTouchingTheService(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"cuerpo vacío", ``},
		{"JSON malformado", `{"email":`},
		{"email ausente", `{"password":"una-contraseña"}`},
		{"email en blanco", `{"email":" ","password":"una-contraseña"}`},
		{"email sin arroba", `{"email":"sin-arroba","password":"una-contraseña"}`},
		{"contraseña ausente", `{"email":"nico@nosepudo.ar"}`},
		{"campo desconocido", `{"email":"nico@nosepudo.ar","password":"x","remember":true}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service := &mockAuthService{}

			status, body := callLogin(t, service, c.body)

			if status != http.StatusBadRequest {
				t.Errorf("status = %d, se esperaba 400: %v", status, body)
			}
			if service.loginCalled != 0 {
				t.Error("el service corrió pese a que el cuerpo era inválido")
			}
		})
	}
}

// A diferencia del alta, el login no le exige largo mínimo a la contraseña.
func TestLoginEndpointAcceptsAShortPassword(t *testing.T) {
	service := &mockAuthService{err: model.ErrInvalidCredentials}

	status, _ := callLogin(t, service, `{"email":"nico@nosepudo.ar","password":"x"}`)

	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401 y no un 400 de validación", status)
	}
	if service.loginCalled != 1 {
		t.Error("la petición no llegó al service")
	}
}
