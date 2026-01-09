package tokenGenerator

import "time"

// TokenClaims - технологически нейтральная структура для JWT claims
type TokenClaims struct {
	Subject   string
	IssuedAt  time.Time
	NotBefore time.Time
	ExpiresAt time.Time
	Extra     map[string]interface{}
}

// ParsedToken - результат парсинга токена
type ParsedToken struct {
	Subject string
	Claims  map[string]interface{}
	Valid   bool
}

// TokenGenerator - интерфейс для генерации и валидации токенов
type TokenGenerator interface {
	// GenerateToken создает подписанный токен
	GenerateToken(claims TokenClaims) (string, error)

	// ParseToken парсит и валидирует токен
	ParseToken(tokenString string) (*ParsedToken, error)
}
