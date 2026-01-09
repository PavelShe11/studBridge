package entity

import "time"

type RefreshTokenSession struct {
	Id           string
	AccountID    string
	RefreshToken string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
