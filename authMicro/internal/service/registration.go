package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	serviceErr "github.com/PavelShe11/studbridge/authMicro/internal/service/error"
	"github.com/PavelShe11/studbridge/authMicro/utlis/generator"
	"github.com/PavelShe11/studbridge/authMicro/utlis/hash"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"
)

type RegisterAnswer struct {
	CodeExpires int64
	CodePattern string
}

type RegistrationService struct {
	registrationSessionRepository port.RegistrationSessionRepository
	accountProvider               port.AccountProvider
	logger                        logger.Logger
	CodeGenConfig                 config.CodeGenConfig
}

func NewRegistrationService(
	registrationSessionRepository port.RegistrationSessionRepository,
	accountProvider port.AccountProvider,
	logger logger.Logger,
	codeGenConfig config.CodeGenConfig,
) *RegistrationService {
	return &RegistrationService{
		registrationSessionRepository: registrationSessionRepository,
		accountProvider:               accountProvider,
		logger:                        logger,
		CodeGenConfig:                 codeGenConfig,
	}
}

func (r *RegistrationService) Register(ctx context.Context, userData map[string]any, lang string) (*RegisterAnswer, error) {
	r.cleanupExpiredSessions(ctx)

	if err := r.accountProvider.ValidateAccountData(ctx, userData, lang); err != nil {
		return nil, err
	}

	email, ok := userData["email"].(string)
	if !ok {
		r.logger.Error(errors.New("email not found in response"))
		return nil, commonEntity.NewInternalError()
	}

	account, err := r.accountProvider.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var session *entity.RegistrationSession

	if account != nil {
		session, err = r.createOrUpdateSession(ctx, email, "")
	} else {
		var plaintextCode string
		plaintextCode, err = generator.Reggen(r.CodeGenConfig.CodePattern, r.CodeGenConfig.CodeMaxLength)
		if err != nil {
			r.logger.Error(err)
			return nil, commonEntity.NewInternalError()
		}
		session, err = r.createOrUpdateSession(ctx, email, plaintextCode)
	}

	if err != nil {
		r.logger.Error(fmt.Errorf("failed to create or update session: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	return &RegisterAnswer{
		CodeExpires: session.CodeExpires.Unix(),
		CodePattern: r.CodeGenConfig.CodePattern,
	}, nil
}

func (r *RegistrationService) ConfirmRegistration(ctx context.Context, userData map[string]any, lang string) error {
	if err := r.accountProvider.ValidateAccountData(ctx, userData, lang); err != nil {
		return err
	}

	email, ok := userData["email"].(string)
	if !ok {
		r.logger.Error(errors.New("email not found in userData"))
		return commonEntity.NewInternalError()
	}

	if err := r.validateConfirmationCode(ctx, email, userData); err != nil {
		return err
	}

	if err := r.accountProvider.CreateAccount(ctx, userData, lang); err != nil {
		return err
	}

	if err := r.registrationSessionRepository.DeleteByEmail(ctx, email); err != nil {
		r.logger.Error(err)
		return commonEntity.NewInternalError()
	}

	r.logger.Info("Account successfully created for email=" + email)
	return nil
}

func (r *RegistrationService) cleanupExpiredSessions(ctx context.Context) {
	if err := r.registrationSessionRepository.CleanExpired(ctx); err != nil {
		r.logger.Error(fmt.Errorf("error cleaning expired registration sessions: %w", err))
	}
}

func (r *RegistrationService) validateConfirmationCode(ctx context.Context, email string, userData map[string]any) error {
	session, err := r.registrationSessionRepository.FindByEmail(ctx, email)
	if err != nil {
		r.logger.Error(err)
		return commonEntity.NewInternalError()
	}
	if session == nil {
		return serviceErr.NewInvalidCodeError()
	}
	if session.CodeExpires.Before(time.Now()) {
		return serviceErr.NewCodeExpiredError()
	}

	submittedCode, ok := userData["code"].(string)
	if !ok || submittedCode == "" || !hash.VerifyCode(session.Code, submittedCode) {
		return serviceErr.NewInvalidCodeError()
	}

	return nil
}

func (r *RegistrationService) createOrUpdateSession(ctx context.Context, email string, code string) (*entity.RegistrationSession, error) {
	session, err := r.registrationSessionRepository.FindByEmail(ctx, email)

	if err != nil {
		r.logger.Error(err)
		return nil, err
	}

	originalCode := code
	if code != "" {
		code, err = hash.HashCode(code)
		if err != nil {
			r.logger.Error(fmt.Errorf("failed to hash verification code: %w", err))
			return nil, commonEntity.NewInternalError()
		}
	}

	if session == nil {
		session = &entity.RegistrationSession{
			Code:        code,
			Email:       email,
			CodeExpires: time.Now().Add(r.CodeGenConfig.CodeTTL),
			CreatedAt:   time.Now(),
		}
	} else if session.CodeExpires.Before(time.Now()) || (session.Code == "" && code != "") {
		session.Code = code
		session.CodeExpires = time.Now().Add(r.CodeGenConfig.CodeTTL)
	} else {
		return session, nil
	}

	if err := r.registrationSessionRepository.Save(ctx, session); err != nil {
		r.logger.Error(err)
		return nil, err
	}

	debugSession := *session
	debugSession.Code = originalCode
	r.logger.Debug(debugSession)

	return session, nil
}
