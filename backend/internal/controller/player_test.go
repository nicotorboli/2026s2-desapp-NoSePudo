package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type mockPlayerService struct {
	err     error
	players []model.Player
}

func (m *mockPlayerService) ListPlayers(ctx context.Context) ([]model.Player, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.players, nil
}

func TestRestPlayerController_GetPlayers_Success(t *testing.T) {
	mockSvc := &mockPlayerService{
		players: []model.Player{
			{ID: 10, Name: "Ernesto Provitillo", Position: 5},
			{ID: 20, Name: "Alfre Montes de Oca", Position: 1},
		},
	}
	c := controller.NewPlayerController(mockSvc)
	endpoint := c.GetPlayers()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	err := endpoint(w, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var raw []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(raw) != 2 {
		t.Fatalf("expected 2 items, got %d", len(raw))
	}

	// Verify ID is NOT exposed in the output
	for i, item := range raw {
		if _, exists := item["id"]; exists {
			t.Errorf("item %d contains 'id' field, but identifier should not be exposed", i)
		}
		if _, exists := item["ID"]; exists {
			t.Errorf("item %d contains 'ID' field, but identifier should not be exposed", i)
		}
		if _, exists := item["name"]; !exists {
			t.Errorf("item %d missing 'name' field", i)
		}
		if _, exists := item["position"]; !exists {
			t.Errorf("item %d missing 'position' field", i)
		}
	}
}

func TestRestPlayerController_GetPlayers_Error(t *testing.T) {
	expectedErr := errors.New("service failure")
	mockSvc := &mockPlayerService{err: expectedErr}
	c := controller.NewPlayerController(mockSvc)
	endpoint := c.GetPlayers()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/players", nil)
	w := httptest.NewRecorder()

	err := endpoint(w, req)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
