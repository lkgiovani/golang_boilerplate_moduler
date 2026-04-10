package securityrepo

import (
	"context"
	"errors"
	"time"

	"golang_boilerplate_module/internal/modules/security/securitydomain"
)

// ErrAlreadyBlocked signals that a user already has an active security block,
// typically surfaced when a unique-index violation occurs on BlockUser.
var ErrAlreadyBlocked = errors.New("security: user already has active block")

// SecurityRepository defines the contract for security monitoring data access.
type SecurityRepository interface {
	CreateSuspiciousActivity(ctx context.Context, activity *securitydomain.SuspiciousActivity) error
	GetRecentByUserID(ctx context.Context, userID int64, since time.Time) ([]securitydomain.SuspiciousActivity, error)
	CountByUserAndSeverity(ctx context.Context, userID int64, severity securitydomain.Severity, since time.Time) (int64, error)
	BlockUser(ctx context.Context, block *securitydomain.UserSecurityBlock) error
	GetActiveBlock(ctx context.Context, userID int64) (*securitydomain.UserSecurityBlock, error)
	UnblockUser(ctx context.Context, userID int64, unblockedBy int64) error
	AutoUnblock(ctx context.Context, userID int64) error
}
