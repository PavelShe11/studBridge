package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"

	"github.com/jmoiron/sqlx"
)

type RegistrationSessionRepository struct {
	db     *sqlx.DB
	getter *trmsql.CtxGetter
}

func NewRegistrationSessionRepository(db *sqlx.DB, getter *trmsql.CtxGetter) *RegistrationSessionRepository {
	return &RegistrationSessionRepository{
		db:     db,
		getter: getter,
	}
}

func (r *RegistrationSessionRepository) FindByEmail(ctx context.Context, email string) (*entity.RegistrationSession, error) {
	query := "SELECT * FROM registration_session WHERE email = $1"
	result := &entity.RegistrationSession{}
	row := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowxContext(ctx, query, email)
	err := row.StructScan(result)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RegistrationSessionRepository) Save(ctx context.Context, session *entity.RegistrationSession) error {
	query := `INSERT INTO registration_session (code, email, code_expires)
	VALUES ($1, $2, $3)
	ON CONFLICT (email)
	DO UPDATE
	SET code = EXCLUDED.code, code_expires = EXCLUDED.code_expires
	RETURNING id, code, email, code_expires, created_at`

	err := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowxContext(ctx, query, session.Code, session.Email, session.CodeExpires).StructScan(session)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegistrationSessionRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := "DELETE FROM registration_session WHERE email = $1"
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query, email)
	if err != nil {
		return err
	}
	return nil
}

func (r *RegistrationSessionRepository) CleanExpired(ctx context.Context) error {
	query := "DELETE FROM registration_session WHERE code_expires < NOW()"
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
