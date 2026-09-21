// Package service contains application services.
package service

import (
	"context"
	"errors"
	"testing"
)

type mockRepo struct {
	err error
}

func (m *mockRepo) Ping(_ context.Context) error {
	return m.err
}

func TestHealthService_CheckHealth(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &mockRepo{err: nil}
		svc := NewHealthService(repo)

		status, err := svc.CheckHealth(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Status != "ok" {
			t.Errorf("expected status 'ok', got '%s'", status.Status)
		}
		if status.Timestamp.IsZero() {
			t.Errorf("expected non-zero timestamp")
		}
	})

	t.Run("RepoFailure", func(t *testing.T) {
		repo := &mockRepo{err: errors.New("connection failed")}
		svc := NewHealthService(repo)

		_, err := svc.CheckHealth(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
