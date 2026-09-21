// Package dto contains HTTP data transfer objects.
package dto

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// HealthResponseDTO represents the HTTP response data transfer object for health checks.
// Per Constitution Principle IV, boundary types have serialization tags and explicit conversion methods.
type HealthResponseDTO struct {
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// DesdeModelo constructs a HealthResponseDTO from the domain HealthStatus.
func (dto HealthResponseDTO) DesdeModelo(m model.HealthStatus) HealthResponseDTO {
	return HealthResponseDTO{
		Status:    m.Status,
		Timestamp: m.Timestamp,
	}
}

// AModelo converts the DTO to the domain HealthStatus model.
func (dto HealthResponseDTO) AModelo() model.HealthStatus {
	return model.HealthStatus{
		Status:    dto.Status,
		Timestamp: dto.Timestamp,
	}
}
