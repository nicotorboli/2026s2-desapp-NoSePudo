package model

import "time"

type AuditLog struct {
	CreatedAt  time.Time
	DiffOld    map[string]any
	DiffNew    map[string]any
	EntityType string
	Action     string
	Actor      string
	ID         int64
	EntityID   int64
}
