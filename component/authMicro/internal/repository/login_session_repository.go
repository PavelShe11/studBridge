package repository

import "github.com/jmoiron/sqlx"

type LoginSessionRepository struct {
	db *sqlx.DB
}
