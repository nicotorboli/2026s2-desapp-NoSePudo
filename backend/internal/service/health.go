// Package service contains application services.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

// HealthService defines the business logic interface for system health.
type HealthService interface {
	CheckHealth(ctx context.Context) (model.HealthStatus, error)
}

type healthService struct {
	repo repository.HealthRepository
}

// NewHealthService creates a new HealthService with injected dependencies.
func NewHealthService(repo repository.HealthRepository) HealthService {
	return &healthService{repo: repo}
}

// CheckHealth coordinates persistence verification and returns the domain HealthStatus.
func (s *healthService) CheckHealth(ctx context.Context) (model.HealthStatus, error) {
	if err := s.repo.Ping(ctx); err != nil {
		return model.HealthStatus{}, fmt.Errorf("repository ping failed: %w", err)
	}

	return model.HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}, nil
}
