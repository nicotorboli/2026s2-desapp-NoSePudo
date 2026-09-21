// Package controller contains HTTP request handlers.
package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type mockService struct {
	status model.HealthStatus
	err    error
}

func (m *mockService) CheckHealth(_ context.Context) (model.HealthStatus, error) {
	return m.status, m.err
}

func TestHealthController_HandleHealth(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		svc := &mockService{
			status: model.HealthStatus{
				Status:    "ok",
				Timestamp: time.Now().UTC(),
			},
			err: nil,
		}
		ctrl := NewHealthController(svc)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		ctrl.HandleHealth(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		svc := &mockService{
			err: errors.New("service failure"),
		}
		ctrl := NewHealthController(svc)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		ctrl.HandleHealth(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError, got %d", rec.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		svc := &mockService{}
		ctrl := NewHealthController(svc)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/health", nil)
		rec := httptest.NewRecorder()

		ctrl.HandleHealth(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 MethodNotAllowed, got %d", rec.Code)
		}
	})
}
