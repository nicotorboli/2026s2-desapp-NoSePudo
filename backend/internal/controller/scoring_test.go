package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type mockScoringService struct {
	getLeagueCoefficientFn func(ctx context.Context, leagueCode string) (dto.LeagueCoefficientResponse, bool, error)
}

func (m *mockScoringService) ListStrategies() []dto.ScoringStrategyResponse { return nil }

func (m *mockScoringService) ActiveStrategy(ctx context.Context) (dto.ActiveStrategyResponse, bool, error) {
	return dto.ActiveStrategyResponse{}, false, nil
}

func (m *mockScoringService) ActivateStrategy(ctx context.Context, strategyID int, actorID int64) error {
	return nil
}

func (m *mockScoringService) SetLeagueCoefficient(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error {
	return nil
}

func (m *mockScoringService) GetLeagueCoefficient(ctx context.Context, leagueCode string) (dto.LeagueCoefficientResponse, bool, error) {
	if m.getLeagueCoefficientFn != nil {
		return m.getLeagueCoefficientFn(ctx, leagueCode)
	}
	return dto.LeagueCoefficientResponse{}, false, nil
}

func (m *mockScoringService) GetScore(ctx context.Context, playerID int64) (dto.PlayerScoreResponse, error) {
	return dto.PlayerScoreResponse{}, nil
}

func (m *mockScoringService) GetScoreHistory(ctx context.Context, playerID int64) ([]dto.PlayerScoreResponse, error) {
	return nil, nil
}

func (m *mockScoringService) ListRanking(ctx context.Context, filter dto.PlayerFilterDTO, sortOrder string, offset int) (dto.PlayerRankingPageResponse, error) {
	return dto.PlayerRankingPageResponse{}, nil
}

func serveLeagueCoefficient(t *testing.T, svc *mockScoringService) dto.LeagueCoefficientResponse {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := httphandler.Wrap(controller.NewScoringController(svc).GetLeagueCoefficient(), logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/scoring/league-coefficients/PL", nil)
	req.SetPathValue("league", "PL")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", recorder.Code, recorder.Body.String())
	}

	var response dto.LeagueCoefficientResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	return response
}

func TestScoringControllerGetLeagueCoefficientFallsBackToDefault(t *testing.T) {
	response := serveLeagueCoefficient(t, &mockScoringService{})

	if response.LeagueCode != "PL" || response.Coefficient != 1.0 {
		t.Errorf("coeficiente por defecto inesperado: %+v", response)
	}
	if response.EffectiveFrom != nil {
		t.Errorf("effectiveFrom debería ser null para el default: %v", *response.EffectiveFrom)
	}
}

func TestScoringControllerGetLeagueCoefficientReturnsStoredValue(t *testing.T) {
	effectiveFrom := "2026-11-02T10:00:00Z"
	response := serveLeagueCoefficient(t, &mockScoringService{
		getLeagueCoefficientFn: func(context.Context, string) (dto.LeagueCoefficientResponse, bool, error) {
			return dto.LeagueCoefficientResponse{
				LeagueCode:    "PL",
				Coefficient:   1.5,
				EffectiveFrom: &effectiveFrom,
			}, true, nil
		},
	})

	if response.Coefficient != 1.5 {
		t.Errorf("coefficient = %v, se esperaba 1.5", response.Coefficient)
	}
	if response.EffectiveFrom == nil || *response.EffectiveFrom != effectiveFrom {
		t.Errorf("effectiveFrom = %v", response.EffectiveFrom)
	}
}

type mockQuoteService struct {
	historyFn func(ctx context.Context, playerID int64) ([]dto.PlayerQuoteResponse, error)
	priceFn   func(ctx context.Context, playerID int64, at time.Time) (dto.PlayerPriceResponse, error)
}

func (m *mockQuoteService) GetQuoteHistory(ctx context.Context, playerID int64) ([]dto.PlayerQuoteResponse, error) {
	if m.historyFn != nil {
		return m.historyFn(ctx, playerID)
	}

	return nil, nil
}

func (m *mockQuoteService) GetPriceAt(ctx context.Context, playerID int64, at time.Time) (dto.PlayerPriceResponse, error) {
	if m.priceFn != nil {
		return m.priceFn(ctx, playerID, at)
	}

	return dto.PlayerPriceResponse{}, nil
}

func newQuoteController(quotes *mockQuoteService) *controller.RestScoringController {
	return controller.NewScoringController(&mockScoringService{}, controller.WithQuoteService(quotes))
}

func serveQuoteEndpoint(t *testing.T, endpoint httphandler.Endpoint, target, id string) *httptest.ResponseRecorder {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := httphandler.Wrap(endpoint, logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if id != "" {
		req.SetPathValue("id", id)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder
}

func TestScoringControllerGetQuoteHistoryOK(t *testing.T) {
	quotes := &mockQuoteService{
		historyFn: func(context.Context, int64) ([]dto.PlayerQuoteResponse, error) {
			return []dto.PlayerQuoteResponse{{Price: "51.0000", PlayerID: 1}}, nil
		},
	}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetQuoteHistory(), "/players/1/quotes", "1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", recorder.Code, recorder.Body.String())
	}

	var items []dto.PlayerQuoteResponse
	if err := json.NewDecoder(recorder.Body).Decode(&items); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
	if len(items) != 1 || items[0].Price != "51.0000" {
		t.Errorf("items = %+v", items)
	}
}

func TestScoringControllerGetQuoteHistoryIDInvalido(t *testing.T) {
	quotes := &mockQuoteService{}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetQuoteHistory(), "/players/abc/quotes", "abc")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, se esperaba 400", recorder.Code)
	}
}

func TestScoringControllerGetQuoteHistoryJugadorInexistente(t *testing.T) {
	quotes := &mockQuoteService{
		historyFn: func(context.Context, int64) ([]dto.PlayerQuoteResponse, error) {
			return nil, service.ErrNotFound
		},
	}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetQuoteHistory(), "/players/1/quotes", "1")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, se esperaba 404", recorder.Code)
	}
}

func TestScoringControllerGetPriceAtOK(t *testing.T) {
	var received time.Time
	quotes := &mockQuoteService{
		priceFn: func(_ context.Context, _ int64, at time.Time) (dto.PlayerPriceResponse, error) {
			received = at
			return dto.PlayerPriceResponse{PlayerID: 1, Price: "51.0000"}, nil
		},
	}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetPriceAt(), "/players/1/price", "1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", recorder.Code, recorder.Body.String())
	}
	if received.IsZero() {
		t.Error("sin date debería reconstruir el precio en el instante actual")
	}
}

func TestScoringControllerGetPriceAtConDate(t *testing.T) {
	var received time.Time
	quotes := &mockQuoteService{
		priceFn: func(_ context.Context, _ int64, at time.Time) (dto.PlayerPriceResponse, error) {
			received = at
			return dto.PlayerPriceResponse{}, nil
		},
	}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetPriceAt(), "/players/1/price?date=2026-11-02T10:00:00Z", "1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %s", recorder.Code, recorder.Body.String())
	}
	if want := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC); !received.Equal(want) {
		t.Errorf("date = %v, se esperaba %v", received, want)
	}
}

func TestScoringControllerGetPriceAtIDInvalido(t *testing.T) {
	quotes := &mockQuoteService{}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetPriceAt(), "/players/abc/price", "abc")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, se esperaba 400", recorder.Code)
	}
}

func TestScoringControllerGetPriceAtDateInvalida(t *testing.T) {
	quotes := &mockQuoteService{}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetPriceAt(), "/players/1/price?date=no-es-fecha", "1")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, se esperaba 400", recorder.Code)
	}
}

func TestScoringControllerGetPriceAtSinScore(t *testing.T) {
	quotes := &mockQuoteService{
		priceFn: func(context.Context, int64, time.Time) (dto.PlayerPriceResponse, error) {
			return dto.PlayerPriceResponse{}, service.ErrScoreNotFound
		},
	}

	recorder := serveQuoteEndpoint(t, newQuoteController(quotes).GetPriceAt(), "/players/1/price", "1")
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, se esperaba 404", recorder.Code)
	}
}
