package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
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

// newTestServer arma el servidor con todos los controllers mockeados. Todo
// caso lo usa, así que agregar un controller se arregla en un solo lugar.
func newTestServer() *server.Server {
	return server.NewServer(
		logger.NewLog(),
		&controller.Container{
			Player: &mockPlayerController{},
			Auth:   &mockAuthController{},
		},
		middleware.NewContainer(),
	)
}

func TestServer_ServeHTTP_GetPlayers(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/unknown-route", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
