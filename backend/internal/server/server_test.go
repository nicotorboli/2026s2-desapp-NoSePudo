package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/server"
)

type mockPlayerController struct{}

func (m *mockPlayerController) GetPlayers() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		return httphandler.Encode(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

type mockAuthController struct{}

func (m *mockAuthController) Register() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		return httphandler.Encode(w, http.StatusCreated, map[string]string{"status": "created"})
	}
}

func (m *mockAuthController) Login() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		return httphandler.Encode(w, http.StatusOK, map[string]string{"status": "signed in"})
	}
}

func (m *mockAuthController) Refresh() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		return httphandler.Encode(w, http.StatusOK, map[string]string{"status": "renewed"})
	}
}

func (m *mockAuthController) Logout() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

// stubTokenVerifier acepta o rechaza sin firmar nada. Lo que estos casos
// prueban es el ruteo y la cadena que se le aplica a cada nivel, no cómo se
// verifica una credencial.
type stubTokenVerifier struct {
	err error
}

func (s *stubTokenVerifier) Verify(string) (adapters.Claims, error) {
	if s.err != nil {
		return adapters.Claims{}, s.err
	}
	return adapters.Claims{Subject: 42, Privilege: model.PrivilegeUser, Kind: adapters.KindAccess}, nil
}

// newTestServer arma el servidor con todos los controllers mockeados. Todo
// caso lo usa, así que agregar un controller se arregla en un solo lugar.
func newTestServer() *server.Server {
	return newTestServerWith(&stubTokenVerifier{})
}

func newTestServerWith(verifier middleware.TokenVerifier) *server.Server {
	return server.NewServer(
		logger.NewLog(),
		&controller.Container{
			Player: &mockPlayerController{},
			Auth: &mockAuthController{},
		},
		middleware.NewContainer(verifier),
	)
}

func TestServer_ServeHTTP_GetPlayers(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	// El catálogo pasó a exigir credencial, así que el caso que lo pedía sin
	// nada ahora la presenta.
	req.Header.Set("Authorization", "Bearer una-credencial")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

// La contracara del caso de arriba: sin credencial el catálogo se niega, que es
// lo que la tabla de rutas declara.
func TestServer_ServeHTTP_GetPlayers_SinCredencial(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// Y si la credencial no verifica, tampoco.
func TestServer_ServeHTTP_GetPlayers_CredencialInvalida(t *testing.T) {
	srv := newTestServerWith(&stubTokenVerifier{err: adapters.ErrTokenInvalid})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	req.Header.Set("Authorization", "Bearer una-credencial-alterada")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// Los endpoints anónimos siguen sirviéndose sin credencial.
func TestServer_ServeHTTP_RutasAnonimas(t *testing.T) {
	cases := []struct {
		pattern string
		want int
	}{
		{"/auth/register", http.StatusCreated},
		{"/auth/login", http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.pattern, func(t *testing.T) {
			srv := newTestServer()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, c.pattern, nil)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)

			if w.Code != c.want {
				t.Fatalf("expected status %d, got %d", c.want, w.Code)
			}
		})
	}
}

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/unknown-route", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

