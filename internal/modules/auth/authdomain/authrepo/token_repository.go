package authrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/auth/authdomain"

	"github.com/google/uuid"
)

// RefreshTokenRepository defines the contract for refresh token storage
// backed by PostgreSQL via GORM.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *authdomain.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*authdomain.RefreshToken, error)
	GetByJTI(ctx context.Context, jti string) (*authdomain.RefreshToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	RevokeByFamilyID(ctx context.Context, familyID uuid.UUID) error
	RevokeByUserID(ctx context.Context, userID int64) error
	GetActiveByUserID(ctx context.Context, userID int64) ([]authdomain.RefreshToken, error)
	DeleteExpired(ctx context.Context) error
}
