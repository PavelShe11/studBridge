package entity

import "time"

type LoginSession struct {
	Id          string
	AccountId   *string
	Email       string
	Code        string // Stores bcrypt hash of verification code
	CodeExpires time.Time
	CreatedAt   time.Time
}

// IsExpired checks if the login code has expired
func (s *LoginSession) IsExpired() bool {
	return time.Now().After(s.CodeExpires)
}
