package helpers

import (
	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/test/mocks"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/stretchr/testify/mock"
)

// NoopLogger implements logger.Logger interface with no-op methods for testing
type NoopLogger struct{}

func NewNoopLogger() logger.Logger {
	return &NoopLogger{}
}

func (l *NoopLogger) Debug(args ...interface{})                 {}
func (l *NoopLogger) Debugf(format string, args ...interface{}) {}
func (l *NoopLogger) Info(args ...interface{})                  {}
func (l *NoopLogger) Infof(format string, args ...interface{})  {}
func (l *NoopLogger) Warn(args ...interface{})                  {}
func (l *NoopLogger) Warnf(format string, args ...interface{})  {}
func (l *NoopLogger) Error(args ...interface{})                 {}
func (l *NoopLogger) Errorf(format string, args ...interface{}) {}
func (l *NoopLogger) Fatal(args ...interface{})                 {}
func (l *NoopLogger) Fatalf(format string, args ...interface{}) {}

// SetupSuccessfulValidation configures mock for successful validation
func SetupSuccessfulValidation(mockProvider *mocks.MockAccountProvider, userData map[string]any, lang string) {
	mockProvider.On("ValidateAccountData", mock.Anything, userData, lang).Return(nil)
}

// SetupSessionFound configures mock for found session
func SetupSessionFound(mockRepo *mocks.MockRegistrationSessionRepository, session *entity.RegistrationSession) {
	mockRepo.On("FindByEmail", mock.Anything, session.Email).Return(session, nil)
}

// SetupSessionNotFound configures mock for non-existent session
func SetupSessionNotFound(mockRepo *mocks.MockRegistrationSessionRepository, email string) {
	mockRepo.On("FindByEmail", mock.Anything, email).Return(nil, nil)
}

// SetupSuccessfulAccountCreation configures mock for successful account creation
func SetupSuccessfulAccountCreation(mockProvider *mocks.MockAccountProvider, userData map[string]any, lang string) {
	mockProvider.On("CreateAccount", mock.Anything, userData, lang).Return(nil)
}

// SetupCleanupExpired configures mock for cleanup
func SetupCleanupExpired(mockRepo *mocks.MockRegistrationSessionRepository) {
	mockRepo.On("CleanExpired", mock.Anything).Return(nil)
}
