package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/jmoiron/sqlx"
)

type refreshTokenSessionRepository struct {
	db     *sqlx.DB
	getter *trmsql.CtxGetter
}

var _ port.RefreshTokenSessionRepository = (*refreshTokenSessionRepository)(nil)

func NewRefreshTokenSessionRepository(db *sqlx.DB, getter *trmsql.CtxGetter) port.RefreshTokenSessionRepository {
	return &refreshTokenSessionRepository{
		db:     db,
		getter: getter,
	}
}

// Save сохраняет новую сессию refresh token
func (r *refreshTokenSessionRepository) Save(
	ctx context.Context,
	session *entity.RefreshTokenSession,
) error {
	query := `
		INSERT INTO refresh_token_session (account_id, refresh_token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	return r.getter.DefaultTrOrDB(ctx, r.db).QueryRowxContext(
		ctx,
		query,
		session.AccountID,
		session.RefreshToken,
		session.ExpiresAt,
	).Scan(&session.Id, &session.CreatedAt)
}

// FindByToken находит сессию по токену
func (r *refreshTokenSessionRepository) FindByToken(
	ctx context.Context,
	token string,
) (*entity.RefreshTokenSession, error) {
	var session entity.RefreshTokenSession
	query := `SELECT * FROM refresh_token_session WHERE refresh_token = $1`
	err := r.getter.DefaultTrOrDB(ctx, r.db).GetContext(ctx, &session, query, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &session, err
}

// DeleteByToken удаляет сессию по токену
func (r *refreshTokenSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_token_session WHERE refresh_token = $1`
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query, token)
	return err
}

// CleanExpired удаляет истекшие сессии
func (r *refreshTokenSessionRepository) CleanExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_token_session WHERE expires_at < NOW()`
	_, err := r.getter.DefaultTrOrDB(ctx, r.db).ExecContext(ctx, query)
	return err
}
