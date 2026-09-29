package server_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/server"
)

type mockPlayerService struct{}

func (m *mockPlayerService) ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
	return dto.PaginatedResponse[dto.PlayerListItemResponse]{
		Items: []dto.PlayerListItemResponse{
			{ID: 1, Name: "Test Player", Club: "Test Club", League: "Test League", Position: "Attacker"},
		},
		Page:       1,
		Limit:      20,
		Total:      1,
		TotalPages: 1,
	}, nil
}

func (m *mockPlayerService) GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
	return dto.PlayerDetailResponse{ID: id, Name: "Test Player"}, nil
}

type mockSyncService struct{}

func (m *mockSyncService) SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
	return dto.SyncPlayersResponse{Status: "success"}, nil
}

func TestServer_ServeHTTP_GetPlayers(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrls := &controller.Container{
		Player: controller.NewPlayerController(&mockPlayerService{}),
		Sync:   controller.NewSyncController(&mockSyncService{}),
	}
	mid := middleware.NewContainer(l)

	srv := server.NewServer("127.0.0.1:8080", l, ctrls, mid)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrls := &controller.Container{
		Player: controller.NewPlayerController(&mockPlayerService{}),
		Sync:   controller.NewSyncController(&mockSyncService{}),
	}
	mid := middleware.NewContainer(l)

	srv := server.NewServer("127.0.0.1:8080", l, ctrls, mid)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/unknown-route", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
