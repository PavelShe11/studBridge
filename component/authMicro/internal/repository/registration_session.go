package repository

import (
	"authMicro/internal/domain"

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
