package authdomain

import "time"

// EmailVerificationToken represents a single-use token for email confirmation.
type EmailVerificationToken struct {
	ID         int64      `json:"id" gorm:"primarykey;autoIncrement"`
	UserID     int64      `json:"user_id" gorm:"not null;index"`
	Email      string     `json:"email" gorm:"not null;size:255;index"`
	Token      string     `json:"token" gorm:"uniqueIndex;not null;size:255"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	Used       bool       `json:"used" gorm:"default:false;not null"`
	CreatedAt  time.Time  `json:"created_at" gorm:"not null"`
}

// PasswordResetToken represents a single-use token for password reset.
type PasswordResetToken struct {
	ID        int64      `json:"id" gorm:"primarykey;autoIncrement"`
	UserID    int64      `json:"user_id" gorm:"not null;index"`
	Email     string     `json:"email" gorm:"not null;size:255;index"`
	Token     string     `json:"token" gorm:"uniqueIndex;not null;size:255"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	Used      bool       `json:"used" gorm:"default:false;not null"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null"`
}
