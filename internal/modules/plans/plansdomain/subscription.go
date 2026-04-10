package plansdomain

import (
	"database/sql"
	"time"
)

// SubscriptionStatus represents the state of a subscription.
type SubscriptionStatus string

const (
	StatusActive     SubscriptionStatus = "active"
	StatusTrialing   SubscriptionStatus = "trialing"
	StatusPastDue    SubscriptionStatus = "past_due"
	StatusCanceled   SubscriptionStatus = "canceled"
	StatusIncomplete SubscriptionStatus = "incomplete"
	StatusExpired    SubscriptionStatus = "expired"
)

// Subscription represents a user's subscription to a plan.
type Subscription struct {
	ID                   int64              `json:"id" gorm:"primarykey;autoIncrement"`
	UserID               int64              `json:"user_id" gorm:"not null"`
	PlanID               int64              `json:"plan_id" gorm:"not null"`
	Status               SubscriptionStatus `json:"status" gorm:"not null;default:active;size:50"`
	StripeSubscriptionID *string            `json:"stripe_subscription_id,omitempty" gorm:"size:255"`
	StripeCustomerID     *string            `json:"stripe_customer_id,omitempty" gorm:"size:255"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" gorm:"not null;default:false"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	TrialEnd             *time.Time         `json:"trial_end,omitempty"`
	CreatedAt            time.Time          `json:"created_at" gorm:"not null"`
	UpdatedAt            sql.NullTime       `json:"updated_at"`

	// Plan relationship for preloading
	Plan *Plan `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
}

// TableName returns the database table name for Subscription.
func (s *Subscription) TableName() string {
	return "subscriptions"
}
