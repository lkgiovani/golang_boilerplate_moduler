package authdomain

import (
	"time"

	"github.com/google/uuid"
)

// TokenClaims holds the claims embedded in a JWT access token.
type TokenClaims struct {
	UserID        int64  `json:"user_id"`
	Email         string `json:"email"`
	Admin         bool   `json:"admin"`
	TokenID       string `json:"jti"`
	EmailVerified bool   `json:"email_verified"`
}

// TokenPair represents an issued access + refresh token pair.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshToken represents a stored refresh token with rotation tracking.
type RefreshToken struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primarykey"`
	UserID      int64      `json:"user_id" gorm:"not null"`
	UserEmail   string     `json:"user_email" gorm:"not null;size:255"`
	DeviceID    string     `json:"device_id" gorm:"not null;size:255"`
	JTI         string     `json:"jti" gorm:"uniqueIndex;not null;size:255"`
	FamilyID    uuid.UUID  `json:"family_id" gorm:"type:uuid;not null;index"`
	TokenHash   string     `json:"-" gorm:"uniqueIndex;not null;size:255"`
	ExpiresAt   time.Time  `json:"expires_at" gorm:"not null"`
	CreatedAt   time.Time  `json:"created_at" gorm:"not null"`
	Used        bool       `json:"used" gorm:"default:false;not null"`
	UsedAt      *time.Time `json:"used_at,omitempty"`
	RotatedFrom *uuid.UUID `json:"rotated_from,omitempty" gorm:"type:uuid"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	UserAgent   *string    `json:"user_agent,omitempty" gorm:"size:500"`
	IPAddress   *string    `json:"ip_address,omitempty" gorm:"size:45"`
}
