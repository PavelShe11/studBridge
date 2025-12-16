package domain

import "time"

type RegistrationSession struct {
	Id          string    `db:"id"`
	Code        string    `db:"code"`
	Email       string    `db:"email"`
	CodeExpires time.Time `db:"code_expires"`
	CreateAt    time.Time `db:"created_at"`
}
