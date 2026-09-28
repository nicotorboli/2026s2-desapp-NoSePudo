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

func TestServer_ServeHTTP_GetPlayers(t *testing.T) {
	log := logger.NewLog()
	ctrls := &controller.Container{
		Player: &mockPlayerController{},
	}
	mid := middleware.NewContainer()

	srv := server.NewServer(log, ctrls, mid)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	log := logger.NewLog()
	ctrls := &controller.Container{
		Player: &mockPlayerController{},
	}
	mid := middleware.NewContainer()

	srv := server.NewServer(log, ctrls, mid)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/unknown-route", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
