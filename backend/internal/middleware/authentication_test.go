package middleware_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// mockTokenVerifier devuelve lo que el caso necesite. Que esté mockeado es lo
// que permite escribir "una credencial alterada" o "del tipo equivocado" sin
// tener que firmar nada: lo que se prueba acá es qué hace el middleware con el
// veredicto, no cómo se verifica una firma.
type mockTokenVerifier struct {
	err    error
	claims adapters.Claims
	calls  int
}

func (m *mockTokenVerifier) Verify(string) (adapters.Claims, error) {
	m.calls++
	return m.claims, m.err
}

// spyEndpoint registra si la operación de abajo llegó a correr, que es lo que
// FR-008 exige: el rechazo ocurre antes de que corra lógica de negocio.
type spyEndpoint struct {
	actor    middleware.Actor
	hadActor bool
	calls    int
}

func (s *spyEndpoint) endpoint() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		s.calls++
		s.actor, s.hadActor = middleware.ActorFromContext(req.Context())
		return httphandler.Encode(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// serve corre el endpoint decorado a través de Wrap, que es como lo ve un cliente.
func serve(t *testing.T, decorator middleware.Decorator, spy *spyEndpoint, authorization string) (int, map[string]string) {
	t.Helper()

	handler := httphandler.Wrap(decorator(spy.endpoint()), slog.New(slog.NewTextHandler(io.Discard, nil)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	handler.ServeHTTP(recorder, request)

	var body map[string]string
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("la respuesta no es JSON: %v", err)
		}
	}

	return recorder.Code, body
}

func validAccessClaims() adapters.Claims {
	return adapters.Claims{
		ID:        "un-jti",
		SessionID: "una-familia",
		Subject:   42,
		Privilege: model.PrivilegeUser,
		Kind:      adapters.KindAccess,
	}
}

// US3, escenario 5: con una credencial válida el catálogo se sirve y la cuenta
// que pidió queda disponible para la operación.
func TestRequireAccessTokenPublishesTheActor(t *testing.T) {
	verifier := &mockTokenVerifier{claims: validAccessClaims()}
	spy := &spyEndpoint{}

	status, _ := serve(t, middleware.NewAuthentication(verifier).RequireAccessToken(), spy, "Bearer una-credencial")

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", status)
	}
	if spy.calls != 1 {
		t.Errorf("la operación corrió %d veces, se esperaba 1", spy.calls)
	}
	if !spy.hadActor {
		t.Fatal("la operación no recibió el actor")
	}
	if spy.actor.ID != 42 {
		t.Errorf("actor.ID = %d, se esperaba 42", spy.actor.ID)
	}
	if spy.actor.Privilege != model.PrivilegeUser {
		t.Errorf("actor.Privilege = %v", spy.actor.Privilege)
	}
	if spy.actor.SessionID != "una-familia" {
		t.Errorf("actor.SessionID = %q", spy.actor.SessionID)
	}
}

// US3, escenario 4: una credencial presentada con una forma inesperada se
// rechaza en vez de intentar interpretarla.
func TestRequireAccessTokenRefusesMalformedHeaders(t *testing.T) {
	cases := []struct {
		name          string
		authorization string
	}{
		{"sin cabecera", ""},
		{"cabecera vacía", " "},
		{"sin esquema", "una-credencial"},
		{"valor vacío", "Bearer"},
		{"valor vacío con espacio", "Bearer "},
		{"esquema desconocido", "Basic una-credencial"},
		{"esquema Token", "Token una-credencial"},
		{"dos tokens", "Bearer una-credencial y-otra"},
		{"sólo el esquema con basura", "Bearer\tuna-credencial"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			verifier := &mockTokenVerifier{claims: validAccessClaims()}
			spy := &spyEndpoint{}

			status, body := serve(t, middleware.NewAuthentication(verifier).RequireAccessToken(), spy, c.authorization)

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", status)
			}
			if body["error"] != "authentication required" {
				t.Errorf("error = %q, se esperaba el mensaje del contrato", body["error"])
			}
			if spy.calls != 0 {
				t.Error("la operación corrió pese a que la credencial era inaceptable (FR-008)")
			}
			if verifier.calls != 0 {
				t.Error("se intentó verificar una cabecera que no tiene la forma esperada")
			}
		})
	}
}

// El esquema es insensible a mayúsculas, como lo define el RFC 7235.
func TestRequireAccessTokenAcceptsTheSchemeInAnyCase(t *testing.T) {
	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr"} {
		t.Run(scheme, func(t *testing.T) {
			verifier := &mockTokenVerifier{claims: validAccessClaims()}
			spy := &spyEndpoint{}

			status, _ := serve(t, middleware.NewAuthentication(verifier).RequireAccessToken(), spy, scheme+" una-credencial")

			if status != http.StatusOK {
				t.Errorf("status = %d con el esquema %q, se esperaba 200", status, scheme)
			}
		})
	}
}

// US3, escenarios 2 y 3: expirada o alterada, el rechazo es el mismo y no se
// confía en nada de lo que la credencial diga.
func TestRequireAccessTokenRefusesWhatTheVerifierRejects(t *testing.T) {
	cases := []struct {
		err  error
		name string
	}{
		{adapters.ErrTokenExpired, "expirada"},
		{adapters.ErrTokenInvalid, "alterada o mal firmada"},
		{errors.New("algo salió mal"), "un fallo cualquiera del verificador"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// El verificador devuelve claims además del error: el middleware no
			// debe usarlas.
			verifier := &mockTokenVerifier{claims: validAccessClaims(), err: c.err}
			spy := &spyEndpoint{}

			status, body := serve(t, middleware.NewAuthentication(verifier).RequireAccessToken(), spy, "Bearer una-credencial")

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", status)
			}
			if body["error"] != "authentication required" {
				t.Errorf("error = %q: la respuesta revela por qué falló", body["error"])
			}
			if spy.calls != 0 {
				t.Error("la operación corrió con una credencial que no verificó")
			}
		})
	}
}

// FR-038: la tabla de decisión de tipo de credencial contra nivel exigido.
// Las dos credenciales no son intercambiables en ninguna dirección.
func TestTokenKindAgainstRequiredLevel(t *testing.T) {
	cases := []struct {
		name      string
		presented adapters.TokenKind
		renewal   bool
		accepted  bool
	}{
		{"acceso a un endpoint autenticado", adapters.KindAccess, false, true},
		{"refresco a un endpoint autenticado", adapters.KindRefresh, false, false},
		{"refresco al endpoint de renovación", adapters.KindRefresh, true, true},
		{"acceso al endpoint de renovación", adapters.KindAccess, true, false},
		{"tipo desconocido a un endpoint autenticado", adapters.KindUnknown, false, false},
		{"tipo desconocido al endpoint de renovación", adapters.KindUnknown, true, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			claims := validAccessClaims()
			claims.Kind = c.presented
			verifier := &mockTokenVerifier{claims: claims}
			spy := &spyEndpoint{}

			authentication := middleware.NewAuthentication(verifier)
			decorator := authentication.RequireAccessToken()
			if c.renewal {
				decorator = authentication.RequireRefreshToken()
			}

			status, _ := serve(t, decorator, spy, "Bearer una-credencial")

			if c.accepted {
				if status != http.StatusOK {
					t.Errorf("status = %d, se esperaba 200", status)
				}
				return
			}
			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", status)
			}
			if spy.calls != 0 {
				t.Error("la operación corrió con una credencial del tipo equivocado")
			}
		})
	}
}

// Sin pasar por autenticación no hay actor, y eso se distingue de haber sido
// autenticado como la cuenta cero.
func TestActorFromContextReportsItsAbsence(t *testing.T) {
	spy := &spyEndpoint{}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	if err := spy.endpoint()(recorder, request); err != nil {
		t.Fatalf("el endpoint devolvió error: %v", err)
	}

	if spy.hadActor {
		t.Error("una petición sin autenticar trae un actor")
	}
	if spy.actor.ID != 0 {
		t.Errorf("actor.ID = %d en una petición sin autenticar", spy.actor.ID)
	}
}

// El privilegio se lee de la credencial ya verificada, y uno que no se
// reconoce llega como insuficiente y no como superusuario (FR-018).
func TestRequireAccessTokenCarriesAnUnknownPrivilegeAsInsufficient(t *testing.T) {
	claims := validAccessClaims()
	claims.Privilege = model.PrivilegeUnknown
	verifier := &mockTokenVerifier{claims: claims}
	spy := &spyEndpoint{}

	status, _ := serve(t, middleware.NewAuthentication(verifier).RequireAccessToken(), spy, "Bearer una-credencial")

	// Autenticar sí: quién es está firmado. Lo que no alcanza es el privilegio,
	// y eso lo decide la autorización, que es otro punto.
	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", status)
	}
	if spy.actor.Privilege != model.PrivilegeUnknown {
		t.Errorf("actor.Privilege = %v, se esperaba PrivilegeUnknown", spy.actor.Privilege)
	}
	if spy.actor.Privilege.Satisfies(model.PrivilegeUser) {
		t.Error("un privilegio irreconocible alcanza para usuario común")
	}
}
