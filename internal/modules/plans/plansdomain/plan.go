package plansdomain

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Plan represents a subscription plan available in the system.
type Plan struct {
	ID              int64           `json:"id" gorm:"primarykey;autoIncrement"`
	Name            string          `json:"name" gorm:"not null"`
	Slug            string          `json:"slug" gorm:"uniqueIndex;not null"`
	Description     *string         `json:"description,omitempty"`
	PriceCents      int64           `json:"price_cents" gorm:"not null;default:0"`
	Currency        string          `json:"currency" gorm:"not null;default:BRL;size:3"`
	BillingInterval string          `json:"billing_interval" gorm:"not null;default:monthly;size:20"`
	Features        json.RawMessage `json:"features" gorm:"type:jsonb;not null;default:'{}'"`
	Active          bool            `json:"active" gorm:"not null;default:true"`
	SortOrder       int             `json:"sort_order" gorm:"not null;default:0"`
	GatewayPriceID  *string         `json:"gateway_price_id,omitempty" gorm:"size:255"`
	GatewayName     string          `json:"gateway_name" gorm:"not null;default:stripe;size:32"`
	CreatedAt       time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt       sql.NullTime    `json:"updated_at"`
}

// TableName returns the database table name for Plan.
func (p *Plan) TableName() string {
	return "plans"
}

// HasFeature checks whether the plan's Features JSONB contains the given feature key with a truthy value.
func (p *Plan) HasFeature(feature string) bool {
	if len(p.Features) == 0 {
		return false
	}

	var features map[string]any
	if err := json.Unmarshal(p.Features, &features); err != nil {
		return false
	}

	val, exists := features[feature]
	if !exists {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return v != "" && v != "false"
	default:
		return val != nil
	}
}
