package authrepo

import (
	"context"

	"golang_boilerplate_module/internal/modules/auth/authdomain"
)

// VerificationTokenRepository defines the contract for email verification
// and password reset token storage backed by PostgreSQL via GORM.
type VerificationTokenRepository interface {
	CreateEmailToken(ctx context.Context, token *authdomain.EmailVerificationToken) error
	GetEmailTokenByToken(ctx context.Context, token string) (*authdomain.EmailVerificationToken, error)
	MarkEmailTokenUsed(ctx context.Context, id int64) error
	CreatePasswordResetToken(ctx context.Context, token *authdomain.PasswordResetToken) error
	GetPasswordResetByToken(ctx context.Context, token string) (*authdomain.PasswordResetToken, error)
	MarkPasswordResetUsed(ctx context.Context, id int64) error
}
