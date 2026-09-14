package model

import "time"

// HealthStatus represents the domain entity for system status.
// Per Constitution Principle IV, domain models do not have serialization tags.
type HealthStatus struct {
	Status    string
	Timestamp time.Time
}
