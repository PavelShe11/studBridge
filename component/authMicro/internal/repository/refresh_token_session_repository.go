package repository

import "github.com/jmoiron/sqlx"

type RefreshTokenSessionRepository struct {
	db *sqlx.DB
}
