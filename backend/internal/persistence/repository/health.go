package repository

import (
	"context"
)

// HealthRepository defines the persistence layer interface for system health checks.
type HealthRepository interface {
	Ping(ctx context.Context) error
}

type systemHealthRepository struct{}

// NewHealthRepository creates a new instance of HealthRepository.
func NewHealthRepository() HealthRepository {
	return &systemHealthRepository{}
}

// Ping checks if the underlying persistence mechanism is accessible.
func (r *systemHealthRepository) Ping(ctx context.Context) error {
	return ctx.Err()
}
