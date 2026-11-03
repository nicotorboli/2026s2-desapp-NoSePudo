package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
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
