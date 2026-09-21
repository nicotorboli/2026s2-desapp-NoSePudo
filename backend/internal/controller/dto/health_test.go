package dto

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestHealthResponseDTO_DesdeModelo(t *testing.T) {
	now := time.Now().UTC()
	m := model.HealthStatus{
		Timestamp: now,
		Status:    "healthy",
	}

	d := HealthResponseDTO{}.DesdeModelo(m)

	if d.Status != m.Status {
		t.Errorf("expected status %s, got %s", m.Status, d.Status)
	}
	if !d.Timestamp.Equal(m.Timestamp) {
		t.Errorf("expected timestamp %v, got %v", m.Timestamp, d.Timestamp)
	}
}

func TestHealthResponseDTO_AModelo(t *testing.T) {
	now := time.Now().UTC()
	d := HealthResponseDTO{
		Timestamp: now,
		Status:    "degraded",
	}

	m := d.AModelo()

	if m.Status != d.Status {
		t.Errorf("expected status %s, got %s", d.Status, m.Status)
	}
	if !m.Timestamp.Equal(d.Timestamp) {
		t.Errorf("expected timestamp %v, got %v", d.Timestamp, m.Timestamp)
	}
}
