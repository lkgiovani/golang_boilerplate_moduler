package repositories

import "context"

type GenericRepository[T any, ID comparable] interface {
	Add(ctx context.Context, entity *T) (*T, error)
	GetByID(ctx context.Context, id ID) (*T, error)
	UpdateByID(ctx context.Context, id ID, updates map[string]any) (*T, error)
	DeleteByID(ctx context.Context, id ID) error
	DeleteAll(ctx context.Context) error
}
