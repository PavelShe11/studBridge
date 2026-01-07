package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"

	"github.com/jmoiron/sqlx"
)

type LoginSessionRepository struct {
	db     *sqlx.DB
	getter *trmsql.CtxGetter
}

func NewLoginSessionRepository(db *sqlx.DB, getter *trmsql.CtxGetter) *LoginSessionRepository {
	return &LoginSessionRepository{
		db:     db,
		getter: getter,
	}
}

func (r *LoginSessionRepository) FindByEmail(ctx context.Context, email string) (*entity.LoginSession, error) {
	query := "SELECT * FROM login_session WHERE email = $1"
	result := &entity.LoginSession{}
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

func (r *LoginSessionRepository) Save(ctx context.Context, session *entity.LoginSession) error {
	query := `INSERT INTO login_session (account_id, email, code, code_expires)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (email)
	DO UPDATE
	SET account_id = EXCLUDED.account_id, code = EXCLUDED.code, code_expires = EXCLUDED.code_expires
	RETURNING id, account_id, email, code, code_expires, created_at`
	err := r.getter.DefaultTrOrDB(ctx, r.db).QueryRowxContext(ctx, query, session.AccountId, session.Email, session.Code, session.CodeExpires).StructScan(session)
	if err != nil {
		return err
	}
	return nil
}

func (r *LoginSessionRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := "DELETE FROM login_session WHERE email = $1"
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query, email)
	if err != nil {
		return err
	}
	return nil
}

func (r *LoginSessionRepository) CleanExpired(ctx context.Context) error {
	query := "DELETE FROM login_session WHERE code_expires < NOW()"
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
