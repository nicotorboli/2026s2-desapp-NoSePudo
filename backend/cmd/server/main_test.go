package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestHealthHandler_Success(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response dto.HealthResponseDTO
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Errorf("expected non-zero timestamp in response")
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status code %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHealthDTO_Conversions(t *testing.T) {
	now := time.Now().UTC()
	m := model.HealthStatus{
		Status:    "healthy",
		Timestamp: now,
	}

	d := dto.HealthResponseDTO{}.DesdeModelo(m)
	if d.Status != m.Status {
		t.Errorf("expected status %s, got %s", m.Status, d.Status)
	}
	if !d.Timestamp.Equal(m.Timestamp) {
		t.Errorf("expected timestamp %v, got %v", m.Timestamp, d.Timestamp)
	}

	back := d.AModelo()
	if back.Status != m.Status {
		t.Errorf("expected status %s, got %s", m.Status, back.Status)
	}
	if !back.Timestamp.Equal(m.Timestamp) {
		t.Errorf("expected timestamp %v, got %v", m.Timestamp, back.Timestamp)
	}
}
