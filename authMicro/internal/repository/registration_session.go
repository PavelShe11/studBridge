package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PavelShe11/studbridge/auth/internal/entity"

	"github.com/jmoiron/sqlx"
)

type RegistrationSessionRepository struct {
	db *sqlx.DB
}

func NewRegistrationSessionRepository(db *sqlx.DB) *RegistrationSessionRepository {
	return &RegistrationSessionRepository{
		db: db,
	}
}

func (r *RegistrationSessionRepository) FindByEmail(email string) (*entity.RegistrationSession, error) {
	query := "SELECT * FROM registration_session WHERE email = $1"
	result := &entity.RegistrationSession{}
	row := r.db.QueryRowx(query, email)
	err := row.StructScan(result)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RegistrationSessionRepository) Save(session *entity.RegistrationSession) error {
	query := `INSERT INTO registration_session (code, email, code_expires)
	VALUES ($1, $2, $3)
	ON CONFLICT (email)
	DO UPDATE
	SET code = EXCLUDED.code, code_expires = EXCLUDED.code_expires
	RETURNING id, code, email, code_expires, created_at`

	err := r.db.QueryRowx(query, session.Code, session.Email, session.CodeExpires).StructScan(session)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegistrationSessionRepository) DeleteByEmail(email string) error {
	query := "DELETE FROM registration_session WHERE email = $1"
	_, err := r.db.Exec(query, email)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegistrationSessionRepository) CleanExpired(ctx context.Context) error {
	query := "DELETE FROM registration_session WHERE code_expires < NOW()"
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
