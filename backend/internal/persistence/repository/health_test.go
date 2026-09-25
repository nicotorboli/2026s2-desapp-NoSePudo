package repository

import (
	"context"
	"errors"
	"testing"
)

func TestSystemHealthRepository_PingWithLiveContext(t *testing.T) {
	healthRepository := NewHealthRepository()

	if err := healthRepository.Ping(context.Background()); err != nil {
		t.Errorf("expected no error for a live context, got %v", err)
	}
}

func TestSystemHealthRepository_PingWithCancelledContext(t *testing.T) {
	healthRepository := NewHealthRepository()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := healthRepository.Ping(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
