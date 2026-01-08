package port

import (
	"context"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
)

// LoginSessionRepository - интерфейс для работы с сессиями входа
type LoginSessionRepository interface {
	// FindByEmail находит сессию входа по email
	FindByEmail(ctx context.Context, email string) (*entity.LoginSession, error)

	// Save сохраняет или обновляет сессию входа
	Save(ctx context.Context, session *entity.LoginSession) error

	// DeleteByEmail удаляет сессию входа по email
	DeleteByEmail(ctx context.Context, email string) error

	// CleanExpired удаляет истекшие сессии входа
	CleanExpired(ctx context.Context) error
}
