package helpers

import (
	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/test/mocks"
	"github.com/stretchr/testify/mock"
)

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
