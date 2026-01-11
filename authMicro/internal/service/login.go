package service

import (
	"context"
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
	"github.com/PavelShe11/studbridge/common/validation"
)

type LoginAnswer struct {
	CodeExpires int64
	CodePattern string
}

type ConfirmLoginEmailAnswer struct {
	accessToken  string
	accessTTL    int
	refreshToken string
	refreshTTL   int
}

type LoginService struct {
	loginSessionRepository port.LoginSessionRepository
	accountProvider        port.AccountProvider
	logger                 logger.Logger
	CodeGenConfig          config.CodeGenConfig
	validator              *validation.Validator
}

func NewLoginService(
	loginSessionRepository port.LoginSessionRepository,
	accountProvider port.AccountProvider,
	logger logger.Logger,
	codeGenConfig config.CodeGenConfig,
	validator *validation.Validator,
) *LoginService {
	return &LoginService{
		loginSessionRepository: loginSessionRepository,
		accountProvider:        accountProvider,
		logger:                 logger,
		CodeGenConfig:          codeGenConfig,
		validator:              validator,
	}
}

func (l *LoginService) cleanupExpiredSessions(ctx context.Context) {
	if err := l.loginSessionRepository.CleanExpired(ctx); err != nil {
		l.logger.Error(fmt.Errorf("error cleaning expired login sessions: %w", err))
	}
}

func (l *LoginService) verifyAccountStillValid(ctx context.Context, email string, expectedAccountId string) (bool, error) {
	account, err := l.accountProvider.GetAccountByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	if account != nil && account.AccountId != "" {
		return account.AccountId == expectedAccountId, nil
	}

	return false, nil
}

func (l *LoginService) createOrUpdateSession(ctx context.Context, email string, accountId *string, code string) (*entity.LoginSession, error) {
	session, err := l.loginSessionRepository.FindByEmail(ctx, email)
	if err != nil {
		l.logger.Error(err)
		return nil, err
	}

	originalCode := code
	if code != "" {
		code, err = hash.HashCode(code)
		if err != nil {
			l.logger.Error(fmt.Errorf("failed to hash verification code: %w", err))
			return nil, commonEntity.NewInternalError()
		}
	}

	if session == nil {
		session = &entity.LoginSession{
			AccountId:   accountId,
			Email:       email,
			Code:        code,
			CodeExpires: time.Now().Add(l.CodeGenConfig.CodeTTL),
			CreatedAt:   time.Now(),
		}
	} else {
		accountIdChanged := (session.AccountId == nil && accountId != nil) ||
			(session.AccountId != nil && accountId == nil) ||
			(session.AccountId != nil && accountId != nil && *session.AccountId != *accountId)

		if session.CodeExpires.Before(time.Now()) || accountIdChanged {
			session.AccountId = accountId
			session.Code = code
			session.CodeExpires = time.Now().Add(l.CodeGenConfig.CodeTTL)
		} else {
			return session, nil
		}
	}

	if err := l.loginSessionRepository.Save(ctx, session); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	debugSession := *session
	debugSession.Code = originalCode
	l.logger.Debug(debugSession)

	return session, nil
}

func (l *LoginService) Login(ctx context.Context, email string) (*LoginAnswer, error) {
	l.cleanupExpiredSessions(ctx)

	errs := commonEntity.NewValidationError()
	l.validator.Var("email", email, "required,email", errs)
	if len(errs.FieldErrors) > 0 {
		return nil, errs
	}

	account, err := l.accountProvider.GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var accountId *string
	if account != nil && account.AccountId != "" {
		accountId = &account.AccountId
	}

	var session *entity.LoginSession

	if accountId != nil {
		plaintextCode, err := generator.Reggen(l.CodeGenConfig.CodePattern, l.CodeGenConfig.CodeMaxLength)
		if err != nil {
			l.logger.Error(err)
			return nil, commonEntity.NewInternalError()
		}

		session, err = l.createOrUpdateSession(ctx, email, accountId, plaintextCode)
		if err != nil {
			l.logger.Error(fmt.Errorf("failed to create or update login session: %w", err))
			return nil, commonEntity.NewInternalError()
		}
	} else {
		session, err = l.createOrUpdateSession(ctx, email, nil, "")
		if err != nil {
			l.logger.Error(err)
			return nil, err
		}
	}

	return &LoginAnswer{
		CodeExpires: session.CodeExpires.Unix(),
		CodePattern: l.CodeGenConfig.CodePattern,
	}, nil
}

func (l *LoginService) validateConfirmLoginData(ctx context.Context, email string, code string) (*string, error) {
	session, err := l.loginSessionRepository.FindByEmail(ctx, email)
	if err != nil {
		l.logger.Error(err)
		return nil, commonEntity.NewInternalError()
	}
	if session == nil {
		return nil, serviceErr.NewInvalidCodeError()
	}
	if session.CodeExpires.Before(time.Now()) {
		return nil, serviceErr.NewCodeExpiredError()
	}

	if code == "" || !hash.VerifyCode(session.Code, code) {
		return nil, serviceErr.NewInvalidCodeError()
	}

	return session.AccountId, nil
}

func (l *LoginService) ConfirmLogin(ctx context.Context, email string, code string) (string, error) {
	errs := commonEntity.NewValidationError()
	l.validator.Var("email", email, "required,email", errs)
	l.validator.Var("code", code, "required", errs)
	if len(errs.FieldErrors) > 0 {
		return "", errs
	}

	session, err := l.loginSessionRepository.FindByEmail(ctx, email)
	if err != nil {
		l.logger.Error(err)
		return "", commonEntity.NewInternalError()
	}
	if session == nil {
		return "", serviceErr.NewInvalidCodeError()
	}
	if session.CodeExpires.Before(time.Now()) {
		return "", serviceErr.NewCodeExpiredError()
	}
	if code == "" || !hash.VerifyCode(session.Code, code) {
		return "", serviceErr.NewInvalidCodeError()
	}

	accountId := session.AccountId

	if accountId == nil {
		return "", serviceErr.NewInvalidCodeError()
	}

	accountStillValid, _ := l.verifyAccountStillValid(ctx, email, *accountId)

	if err := l.loginSessionRepository.DeleteByEmail(ctx, email); err != nil {
		l.logger.Error(fmt.Errorf("failed to delete session after account switch: %w", err))
		return "", commonEntity.NewInternalError()
	}

	if !accountStillValid {
		l.logger.Info(fmt.Sprintf("Account switch detected for email %s", email))
		return "", serviceErr.NewInvalidCodeError()
	}

	return *accountId, nil
}
