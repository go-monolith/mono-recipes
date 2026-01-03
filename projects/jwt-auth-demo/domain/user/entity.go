package user

import (
	"time"
)

// User represents a user entity in the system.
type User struct {
	ID           string `gorm:"primaryKey;type:text"`
	Email        string `gorm:"uniqueIndex;not null;type:text"`
	PasswordHash string `gorm:"not null;type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName returns the table name for the User entity.
func (User) TableName() string {
	return "users"
}

// TokenPair represents access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Claims represents JWT claims.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}
