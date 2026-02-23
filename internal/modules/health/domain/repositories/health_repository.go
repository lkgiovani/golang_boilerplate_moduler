package repositories

import "context"

type HealthRepository interface {
	Ping(ctx context.Context) (bool, error)
}
