package plansdomain

import (
	"encoding/json"
	"time"
)

// PaymentEvent represents a payment gateway event log entry for idempotency.
type PaymentEvent struct {
	ID             int64           `json:"id" gorm:"primarykey;autoIncrement"`
	StripeEventID  string          `json:"stripe_event_id" gorm:"uniqueIndex;not null;size:255"`
	EventType      string          `json:"event_type" gorm:"not null;size:100"`
	SubscriptionID *int64          `json:"subscription_id,omitempty"`
	UserID         *int64          `json:"user_id,omitempty"`
	Payload        json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	Processed      bool            `json:"processed" gorm:"not null;default:false"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
	Error          *string         `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"created_at" gorm:"not null"`
}

// TableName returns the database table name for PaymentEvent.
func (pe *PaymentEvent) TableName() string {
	return "payment_events"
}
