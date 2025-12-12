package repository

import "github.com/jmoiron/sqlx"

type RegistrationSessionRepository struct {
	db *sqlx.DB
}
