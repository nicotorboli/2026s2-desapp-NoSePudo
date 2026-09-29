package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type mockPlayerSyncService struct {
	syncPlayersFn func(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)
}

func (m *mockPlayerSyncService) SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
	if m.syncPlayersFn != nil {
		return m.syncPlayersFn(ctx, actor)
	}
	return dto.SyncPlayersResponse{}, nil
}

func TestSyncController_SyncPlayers_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerSyncService{
		syncPlayersFn: func(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
			return dto.SyncPlayersResponse{
				Status:           "success",
				Message:          "Player synchronization completed successfully",
				TotalProcessed:   2500,
				TotalUpdated:     45,
				TotalDeactivated: 12,
			}, nil
		},
	}

	c := controller.NewSyncController(mockSvc)
	handler := httphandler.Wrap(c.SyncPlayers(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp dto.SyncPlayersResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalProcessed != 2500 || resp.TotalUpdated != 45 || resp.TotalDeactivated != 12 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSyncController_SyncPlayers_RateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerSyncService{
		syncPlayersFn: func(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
			return dto.SyncPlayersResponse{}, service.ErrRateLimitExceeded
		},
	}

	c := controller.NewSyncController(mockSvc)
	handler := httphandler.Wrap(c.SyncPlayers(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}
}

func TestSyncController_SyncPlayers_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerSyncService{
		syncPlayersFn: func(ctx context.Context, actor string) (dto.SyncPlayersResponse, error) {
			return dto.SyncPlayersResponse{}, errors.New("database connection lost")
		},
	}

	c := controller.NewSyncController(mockSvc)
	handler := httphandler.Wrap(c.SyncPlayers(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}
