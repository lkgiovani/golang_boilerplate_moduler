package securitydomain

import (
	"encoding/json"
	"time"
)

type ActivityType string

const (
	ActivityRateLimitExceeded  ActivityType = "RATE_LIMIT_EXCEEDED"
	ActivityMassCreation       ActivityType = "MASS_CREATION"
	ActivityPatternAbuse       ActivityType = "PATTERN_ABUSE"
	ActivityInvalidData        ActivityType = "INVALID_DATA_ATTEMPTS"
	ActivityUnauthorizedAccess ActivityType = "UNAUTHORIZED_ACCESS"
	ActivitySuspiciousPattern  ActivityType = "SUSPICIOUS_PATTERN"
	ActivityAutomatedBehavior  ActivityType = "AUTOMATED_BEHAVIOR"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// SuspiciousActivity records suspicious user behavior for analysis and automatic blocking.
type SuspiciousActivity struct {
	ID           int64           `json:"id" gorm:"primarykey;autoIncrement"`
	UserID       int64           `json:"user_id" gorm:"not null;index"`
	ActivityType ActivityType    `json:"activity_type" gorm:"not null;size:50"`
	Endpoint     string          `json:"endpoint" gorm:"not null;size:500"`
	IPAddress    *string         `json:"ip_address,omitempty" gorm:"size:45"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	RequestCount int             `json:"request_count" gorm:"default:1"`
	Details      json.RawMessage `json:"details,omitempty" gorm:"type:jsonb"`
	Severity     Severity        `json:"severity" gorm:"not null;default:LOW;size:20"`
	CreatedAt    time.Time       `json:"created_at" gorm:"not null"`
}
