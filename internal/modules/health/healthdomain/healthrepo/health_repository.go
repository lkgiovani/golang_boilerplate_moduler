package healthrepo

import "context"

type HealthRepository interface {
	Ping(ctx context.Context) (bool, error)
	PingRedis(ctx context.Context) bool
}
