package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type mockStatsService struct {
	getPlayerStatisticsFn func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error)
}

func (m *mockStatsService) GetPlayerStatistics(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
	if m.getPlayerStatisticsFn != nil {
		return m.getPlayerStatisticsFn(ctx, id)
	}
	return nil, nil
}

type mockSyncService struct {
	triggerFn        func(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error)
	getRunFn         func(ctx context.Context, id int64) (dto.SyncRunResponse, error)
	listRunsFn       func(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error)
	listSchedulesFn  func(ctx context.Context) ([]dto.SyncScheduleResponse, error)
	updateScheduleFn func(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error)
}

func (m *mockSyncService) Trigger(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error) {
	if m.triggerFn != nil {
		return m.triggerFn(ctx, typ, actorID)
	}
	return dto.SyncRunResponse{}, nil
}

func (m *mockSyncService) GetRun(ctx context.Context, id int64) (dto.SyncRunResponse, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return dto.SyncRunResponse{}, nil
}

func (m *mockSyncService) ListRuns(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error) {
	if m.listRunsFn != nil {
		return m.listRunsFn(ctx, typ, offset, limit)
	}
	return dto.SyncRunPageResponse{}, nil
}

func (m *mockSyncService) ListSchedules(ctx context.Context) ([]dto.SyncScheduleResponse, error) {
	if m.listSchedulesFn != nil {
		return m.listSchedulesFn(ctx)
	}
	return nil, nil
}

func (m *mockSyncService) UpdateSchedule(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error) {
	if m.updateScheduleFn != nil {
		return m.updateScheduleFn(ctx, typ, enabled, intervalMinutes, actorID)
	}
	return dto.SyncScheduleResponse{}, nil
}

func newSyncController(stats *mockStatsService, sync *mockSyncService) *controller.RestSyncController {
	return controller.NewSyncController(stats, sync)
}

func authenticatedContext(t *testing.T) context.Context {
	t.Helper()
	return middleware.WithActor(t.Context(), middleware.Actor{ID: 42})
}

func TestSyncController_CreateRun_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var gotType model.SyncType
	var gotActor *int64
	mockSync := &mockSyncService{
		triggerFn: func(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error) {
			gotType = typ
			gotActor = actorID
			return dto.SyncRunResponse{ID: 7, Type: "players", Trigger: "manual", Status: "running"}, nil
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	handler := httphandler.Wrap(c.CreateRun(), logger)

	req := httptest.NewRequestWithContext(authenticatedContext(t), http.MethodPost, "/sync/players/runs", nil)
	req.SetPathValue("type", "players")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	var resp dto.SyncRunResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != 7 || resp.Status != "running" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if gotType != model.SyncTypePlayers {
		t.Errorf("type = %q, se esperaba players", gotType)
	}
	if gotActor == nil || *gotActor != 42 {
		t.Errorf("actor = %v, se esperaba 42", gotActor)
	}
}

func TestSyncController_CreateRun_InvalidType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := newSyncController(&mockStatsService{}, &mockSyncService{})
	handler := httphandler.Wrap(c.CreateRun(), logger)

	req := httptest.NewRequestWithContext(authenticatedContext(t), http.MethodPost, "/sync/foo/runs", nil)
	req.SetPathValue("type", "foo")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSyncController_GetRun_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSync := &mockSyncService{
		getRunFn: func(ctx context.Context, id int64) (dto.SyncRunResponse, error) {
			if id != 7 {
				t.Fatalf("expected id 7, got %d", id)
			}
			return dto.SyncRunResponse{ID: 7, Status: "succeeded"}, nil
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	mux := http.NewServeMux()
	mux.Handle("GET /sync/runs/{id}", httphandler.Wrap(c.GetRun(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/runs/7", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestSyncController_GetRun_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSync := &mockSyncService{
		getRunFn: func(ctx context.Context, id int64) (dto.SyncRunResponse, error) {
			return dto.SyncRunResponse{}, service.ErrSyncRunNotFound
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	mux := http.NewServeMux()
	mux.Handle("GET /sync/runs/{id}", httphandler.Wrap(c.GetRun(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/runs/999", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestSyncController_GetRun_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := newSyncController(&mockStatsService{}, &mockSyncService{})
	mux := http.NewServeMux()
	mux.Handle("GET /sync/runs/{id}", httphandler.Wrap(c.GetRun(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/runs/abc", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSyncController_ListRuns_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSync := &mockSyncService{
		listRunsFn: func(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error) {
			if typ == nil || *typ != model.SyncTypePlayers {
				t.Errorf("type = %v, se esperaba players", typ)
			}
			if offset != 0 || limit != 20 {
				t.Errorf("offset/limit = %d/%d, se esperaba 0/20", offset, limit)
			}
			return dto.SyncRunPageResponse{Items: []dto.SyncRunResponse{}, Offset: 0, Limit: 20, Total: 0}, nil
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	mux := http.NewServeMux()
	mux.Handle("GET /sync/runs", httphandler.Wrap(c.ListRuns(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/runs?type=players", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestSyncController_ListRuns_InvalidType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := newSyncController(&mockStatsService{}, &mockSyncService{})
	mux := http.NewServeMux()
	mux.Handle("GET /sync/runs", httphandler.Wrap(c.ListRuns(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/runs?type=foo", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSyncController_ListSchedules_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSync := &mockSyncService{
		listSchedulesFn: func(ctx context.Context) ([]dto.SyncScheduleResponse, error) {
			return []dto.SyncScheduleResponse{{Type: "players", Enabled: true, IntervalMinutes: 1440}}, nil
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	mux := http.NewServeMux()
	mux.Handle("GET /sync/schedules", httphandler.Wrap(c.ListSchedules(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sync/schedules", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp []dto.SyncScheduleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 1 || !resp[0].Enabled {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSyncController_UpdateSchedule_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockSync := &mockSyncService{
		updateScheduleFn: func(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error) {
			if typ != model.SyncTypePlayers || !enabled || intervalMinutes != 1440 || actorID != 42 {
				t.Errorf("unexpected args: %v %v %d %d", typ, enabled, intervalMinutes, actorID)
			}
			return dto.SyncScheduleResponse{Type: "players", Enabled: true, IntervalMinutes: 1440}, nil
		},
	}

	c := newSyncController(&mockStatsService{}, mockSync)
	mux := http.NewServeMux()
	mux.Handle("PUT /sync/schedules/{type}", httphandler.Wrap(c.UpdateSchedule(), logger))

	req := httptest.NewRequestWithContext(authenticatedContext(t), http.MethodPut, "/sync/schedules/players", nil)
	req.SetPathValue("type", "players")
	req.Body = io.NopCloser(strings.NewReader(`{"enabled":true,"intervalMinutes":1440}`))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSyncController_UpdateSchedule_InvalidInterval(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	c := newSyncController(&mockStatsService{}, &mockSyncService{})
	mux := http.NewServeMux()
	mux.Handle("PUT /sync/schedules/{type}", httphandler.Wrap(c.UpdateSchedule(), logger))

	req := httptest.NewRequestWithContext(authenticatedContext(t), http.MethodPut, "/sync/schedules/players", nil)
	req.SetPathValue("type", "players")
	req.Body = io.NopCloser(strings.NewReader(`{"enabled":true,"intervalMinutes":1}`))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSyncController_GetPlayerStatistics_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockStatsService{
		getPlayerStatisticsFn: func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
			if id != 42 {
				t.Fatalf("expected id 42, got %d", id)
			}
			return []dto.PlayerStatisticsResponse{
				{Goals: 11, Assists: 6, Shots: 2.8, KeyPasses: 2.0, Dribbles: 2.3, Tackles: 1.2, Interceptions: 0.7, Fouls: 0.8, Clears: 0.5, Blocks: 0.3, WasDribbled: 1.0, Rating: 8.1},
			}, nil
		},
	}

	c := newSyncController(mockSvc, &mockSyncService{})
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
	mockSvc := &mockStatsService{
		getPlayerStatisticsFn: func(ctx context.Context, id int64) ([]dto.PlayerStatisticsResponse, error) {
			return nil, service.ErrNotFound
		},
	}

	c := newSyncController(mockSvc, &mockSyncService{})
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

	c := newSyncController(&mockStatsService{}, &mockSyncService{})
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}/statistics", httphandler.Wrap(c.GetPlayerStatistics(), logger))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players/abc/statistics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
