package port

import (
	"context"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
)

// RefreshTokenSessionRepository - интерфейс для работы с сессиями refresh токенов
type RefreshTokenSessionRepository interface {
	// Save сохраняет новую сессию refresh токена
	Save(ctx context.Context, session *entity.RefreshTokenSession) error

	// FindByToken находит сессию по токену
	FindByToken(ctx context.Context, token string) (*entity.RefreshTokenSession, error)

	// DeleteByToken удаляет сессию по токену
	DeleteByToken(ctx context.Context, token string) error

	// CleanExpired удаляет истекшие сессии
	CleanExpired(ctx context.Context) error
}
