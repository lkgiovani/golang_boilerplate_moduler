package usersdomain

import (
	"database/sql"
	"encoding/json"
	"time"
)

type User struct {
	ID            int64           `json:"id" gorm:"primarykey;autoIncrement"`
	Name          string          `json:"name" gorm:"not null"`
	Email         string          `json:"email" gorm:"uniqueIndex;not null"`
	Password      *string         `json:"-" gorm:"size:255"`
	ImgURL        *string         `json:"img_url" gorm:"column:img_url;size:500"`
	Admin         bool            `json:"admin" gorm:"default:false;not null"`
	Active        bool            `json:"active" gorm:"default:true;not null"`
	EmailVerified bool            `json:"email_verified" gorm:"default:false;not null"`
	Source        string          `json:"source" gorm:"default:LOCAL;not null;size:50"`
	Metadata      json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	LastAccess    *time.Time      `json:"last_access,omitempty"`
	CreatedAt     time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt     sql.NullTime    `json:"updated_at"`
}
