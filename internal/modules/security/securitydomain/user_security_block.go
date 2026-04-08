package securitydomain

import "time"

// UserSecurityBlock records temporary or permanent security blocks on user accounts.
type UserSecurityBlock struct {
	ID                      int64      `json:"id" gorm:"primarykey;autoIncrement"`
	UserID                  int64      `json:"user_id" gorm:"not null;index"`
	Reason                  string     `json:"reason" gorm:"not null;size:500"`
	SuspiciousActivityCount int        `json:"suspicious_activity_count" gorm:"not null;default:0"`
	BlockedAt               time.Time  `json:"blocked_at" gorm:"not null"`
	BlockedUntil            *time.Time `json:"blocked_until,omitempty"`
	UnblockedAt             *time.Time `json:"unblocked_at,omitempty"`
	UnblockedBy             *int64     `json:"unblocked_by,omitempty"`
	Notes                   *string    `json:"notes,omitempty"`
}
