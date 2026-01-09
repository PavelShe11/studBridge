package entity

import "time"

type RegistrationSession struct {
	Id          string
	Code        string
	Email       string
	CodeExpires time.Time
	CreatedAt   time.Time
}

// IsExpired checks if the registration code has expired
func (s *RegistrationSession) IsExpired() bool {
	return time.Now().After(s.CodeExpires)
}

// IsCodeValid validates the code and expiration
func (s *RegistrationSession) IsCodeValid(code string) bool {
	return !s.IsExpired() && s.Code == code
}
