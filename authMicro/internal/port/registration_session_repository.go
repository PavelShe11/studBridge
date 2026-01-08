package port

import (
	"context"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
)

// RegistrationSessionRepository - интерфейс для работы с сессиями регистрации
type RegistrationSessionRepository interface {
	// FindByEmail находит сессию регистрации по email
	FindByEmail(ctx context.Context, email string) (*entity.RegistrationSession, error)

	// Save сохраняет или обновляет сессию регистрации
	Save(ctx context.Context, session *entity.RegistrationSession) error

	// DeleteByEmail удаляет сессию регистрации по email
	DeleteByEmail(ctx context.Context, email string) error

	// CleanExpired удаляет истекшие сессии регистрации
	CleanExpired(ctx context.Context) error
}
