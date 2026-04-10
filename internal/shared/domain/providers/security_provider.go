package providers

import (
	"context"
	"time"
)

// BlockStatus reports whether a user is currently under a security block.
type BlockStatus struct {
	Blocked      bool
	Reason       string
	BlockedUntil *time.Time // nil = permanent or not blocked
}

// BlockChecker abstracts the security module's block lookup so that other
// modules (notably auth middleware) can enforce blocks without importing the
// security module directly.
type BlockChecker interface {
	IsBlocked(ctx context.Context, userID int64) (BlockStatus, error)
}
