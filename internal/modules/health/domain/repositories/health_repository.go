package repositories

import "context"

// HealthRepository defines the contract for health check persistence operations.
type HealthRepository interface {
	Ping(ctx context.Context) (bool, error)
}
