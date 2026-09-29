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

type mockPlayerService struct {
	listPlayersFn   func(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error)
	getPlayerByIDFn func(ctx context.Context, id int64) (dto.PlayerDetailResponse, error)
}

func (m *mockPlayerService) ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
	if m.listPlayersFn != nil {
		return m.listPlayersFn(ctx, filter)
	}
	return dto.PaginatedResponse[dto.PlayerListItemResponse]{}, nil
}

func (m *mockPlayerService) GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
	if m.getPlayerByIDFn != nil {
		return m.getPlayerByIDFn(ctx, id)
	}
	return dto.PlayerDetailResponse{}, nil
}

func TestPlayerController_GetPlayers_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{
		listPlayersFn: func(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
			return dto.PaginatedResponse[dto.PlayerListItemResponse]{
				Items: []dto.PlayerListItemResponse{
					{ID: 1, Name: "Ernesto Provitillo", Club: "Arsenal", League: "Premier League", Position: "Midfielder"},
				},
				Page:       1,
				Limit:      20,
				Total:      1,
				TotalPages: 1,
			}, nil
		},
	}

	c := controller.NewPlayerController(mockSvc)
	handler := httphandler.Wrap(c.GetPlayers, logger)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players?page=1&limit=20", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp dto.PaginatedResponse[dto.PlayerListItemResponse]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 1 || resp.Items[0].Name != "Ernesto Provitillo" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPlayerController_GetPlayers_InvalidQuery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{}
	c := controller.NewPlayerController(mockSvc)
	handler := httphandler.Wrap(c.GetPlayers, logger)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players?page=invalid", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPlayerController_GetPlayerByID_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{
		getPlayerByIDFn: func(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
			if id == 7 {
				return dto.PlayerDetailResponse{
					ID:         7,
					ExternalID: 7821,
					Name:       "Bukayo Saka",
					Club:       "Arsenal FC",
					League:     "Premier League",
					LeagueCode: "PL",
					Position:   "Attacker",
					Active:     true,
				}, nil
			}
			return dto.PlayerDetailResponse{}, service.ErrNotFound
		},
	}

	c := controller.NewPlayerController(mockSvc)

	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}", httphandler.Wrap(c.GetPlayerByID, logger))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players/7", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp dto.PlayerDetailResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Bukayo Saka" || resp.ID != 7 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPlayerController_GetPlayerByID_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{
		getPlayerByIDFn: func(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
			return dto.PlayerDetailResponse{}, service.ErrNotFound
		},
	}

	c := controller.NewPlayerController(mockSvc)
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}", httphandler.Wrap(c.GetPlayerByID, logger))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players/999", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestPlayerController_GetPlayerByID_InvalidID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{}
	c := controller.NewPlayerController(mockSvc)
	mux := http.NewServeMux()
	mux.Handle("GET /players/{id}", httphandler.Wrap(c.GetPlayerByID, logger))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players/abc", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPlayerController_GetPlayers_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSvc := &mockPlayerService{
		listPlayersFn: func(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
			return dto.PaginatedResponse[dto.PlayerListItemResponse]{}, errors.New("db error")
		},
	}

	c := controller.NewPlayerController(mockSvc)
	handler := httphandler.Wrap(c.GetPlayers, logger)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}
