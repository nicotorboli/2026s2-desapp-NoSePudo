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

type mockPlayerStatisticsService struct {
	getPlayerStatisticsFn func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error)
	syncStatisticsFn      func(ctx context.Context) (dto.SyncStatisticsResponse, error)
}

func (m *mockPlayerStatisticsService) GetPlayerStatistics(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
	if m.getPlayerStatisticsFn != nil {
		return m.getPlayerStatisticsFn(ctx, id)
	}
	return nil, nil
}

func (m *mockPlayerStatisticsService) SyncStatistics(ctx context.Context) (dto.SyncStatisticsResponse, error) {
	if m.syncStatisticsFn != nil {
		return m.syncStatisticsFn(ctx)
	}
	return dto.SyncStatisticsResponse{}, nil
}

func newSyncController(players *mockPlayerSyncService, stats *mockPlayerStatisticsService) *controller.RestSyncController {
	return controller.NewSyncController(players, stats)
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

	c := newSyncController(mockSvc, &mockPlayerStatisticsService{})
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

	c := newSyncController(mockSvc, &mockPlayerStatisticsService{})
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

	c := newSyncController(mockSvc, &mockPlayerStatisticsService{})
	handler := httphandler.Wrap(c.SyncPlayers(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestSyncController_SyncStatistics_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerStatisticsService{
		syncStatisticsFn: func(ctx context.Context) (dto.SyncStatisticsResponse, error) {
			return dto.SyncStatisticsResponse{
				Status:         "success",
				Message:        "Player statistics synchronization completed successfully",
				TotalProcessed: 2223,
				TotalMatched:   1878,
				TotalUnmatched: 345,
				TotalInserted:  1878,
			}, nil
		},
	}

	c := newSyncController(&mockPlayerSyncService{}, mockSvc)
	handler := httphandler.Wrap(c.SyncStatistics(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/statistics/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp dto.SyncStatisticsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TotalMatched != 1878 || resp.TotalInserted != 1878 || resp.TotalUnmatched != 345 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSyncController_SyncStatistics_RateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerStatisticsService{
		syncStatisticsFn: func(ctx context.Context) (dto.SyncStatisticsResponse, error) {
			return dto.SyncStatisticsResponse{}, service.ErrStatisticsRateLimitExceeded
		},
	}

	c := newSyncController(&mockPlayerSyncService{}, mockSvc)
	handler := httphandler.Wrap(c.SyncStatistics(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/statistics/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}
}

func TestSyncController_SyncStatistics_Blocked(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerStatisticsService{
		syncStatisticsFn: func(ctx context.Context) (dto.SyncStatisticsResponse, error) {
			return dto.SyncStatisticsResponse{}, service.ErrStatisticsProviderBlocked
		},
	}

	c := newSyncController(&mockPlayerSyncService{}, mockSvc)
	handler := httphandler.Wrap(c.SyncStatistics(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/players/statistics/sync", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rr.Code)
	}
}

func TestSyncController_GetPlayerStatistics_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerStatisticsService{
		getPlayerStatisticsFn: func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
			if id != 42 {
				t.Fatalf("expected id 42, got %d", id)
			}
			return []dto.PlayerStatisticsResponse{
				{Goals: 11, Assists: 6, Shots: 2.8, KeyPasses: 2.0, Dribbles: 2.3, Tackles: 1.2, Interceptions: 0.7, Fouls: 0.8, Clears: 0.5, Blocks: 0.3, WasDribbled: 1.0, Rating: 8.1},
			}, nil
		},
	}

	c := newSyncController(&mockPlayerSyncService{}, mockSvc)
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}/statistics", httphandler.Wrap(c.GetPlayerStatistics(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players/42/statistics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp []dto.PlayerStatisticsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Goals != 11 || resp[0].Rating != 8.1 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSyncController_GetPlayerStatistics_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerStatisticsService{
		getPlayerStatisticsFn: func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
			return nil, service.ErrNotFound
		},
	}

	c := newSyncController(&mockPlayerSyncService{}, mockSvc)
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}/statistics", httphandler.Wrap(c.GetPlayerStatistics(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players/999/statistics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestSyncController_GetPlayerStatistics_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := newSyncController(&mockPlayerSyncService{}, &mockPlayerStatisticsService{})
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}/statistics", httphandler.Wrap(c.GetPlayerStatistics(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players/abc/statistics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
