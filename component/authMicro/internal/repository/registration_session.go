package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/PavelShe11/studbridge/auth/internal/domain"

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

func (r RegistrationSessionRepository) FindByEmail(email string) (*domain.RegistrationSession, error) {
	query := "SELECT * FROM registration_session WHERE email = $1"
	result := &domain.RegistrationSession{}
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

func (r RegistrationSessionRepository) Save(session *domain.RegistrationSession) error {
	query := `INSERT INTO registration_session (code, email, code_expires) 
	VALUES (:code, :email, :code_expires) 
	ON CONFLICT (email) 
	DO UPDATE 
	SET code = EXCLUDED.code, code_expires = EXCLUDED.code_expires`

	_, err := r.db.NamedExec(query, session)
	if err != nil {
		return err
	}
	return nil
}

func (r RegistrationSessionRepository) DeleteByEmail(email string) error {
	query := "DELETE FROM registration_session WHERE email = $1"
	_, err := r.db.Exec(query, email)
	if err != nil {
		return err
	}
	return nil
}

func (r RegistrationSessionRepository) CleanExpired(ctx context.Context) error {
	query := "DELETE FROM registration_session WHERE code_expires < NOW()"
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
