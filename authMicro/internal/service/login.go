package service

import (
	"context"
	"fmt"
	"time"

	"github.com/PavelShe11/studbridge/auth/internal/config"
	"github.com/PavelShe11/studbridge/auth/internal/entity"
	"github.com/PavelShe11/studbridge/auth/internal/repository"
	"github.com/PavelShe11/studbridge/auth/utlis/generator"
	"github.com/PavelShe11/studbridge/auth/utlis/hash"
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/validation"
	"google.golang.org/grpc/status"
)

type LoginAnswer struct {
	CodeExpires int64  `json:"code_expires"`
	CodePattern string `json:"code_pattern"`
}

type ConfirmLoginEmailAnswer struct {
	accessToken  string
	accessTTL    int
	refreshToken string
	refreshTTL   int
}

type LoginService struct {
	loginSessionRepository *repository.LoginSessionRepository
	accountService         grpcApi.AccountServiceClient
	logger                 logger.Logger
	CodeGenConfig          config.CodeGenConfig
	validator              *validation.Validator
}

func NewLoginService(
	loginSessionRepository *repository.LoginSessionRepository,
	accountService grpcApi.AccountServiceClient,
	logger logger.Logger,
	codeGenConfig config.CodeGenConfig,
	validator *validation.Validator,
) *LoginService {
	return &LoginService{
		loginSessionRepository: loginSessionRepository,
		accountService:         accountService,
		logger:                 logger,
		CodeGenConfig:          codeGenConfig,
		validator:              validator,
	}
}

func (l *LoginService) cleanupExpiredSessions() {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCleanup()
	if err := l.loginSessionRepository.CleanExpired(cleanupCtx); err != nil {
		l.logger.Error(fmt.Errorf("error cleaning expired login sessions: %w", err))
	}
}

func (l *LoginService) getAccountByEmail(email string) (*grpcApi.GetAccountResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accountGrpc, err := l.accountService.GetAccountByEmail(
		ctx,
		&grpcApi.GetAccountByEmailRequest{Email: email},
	)

	if err != nil {
		st, _ := status.FromError(err)
		l.logger.Error(fmt.Errorf("GetAccountByEmail error: %v, grpc status: %v", err, st))
		return nil, commonEntity.NewInternalError()
	}
	return accountGrpc, nil
}

func (l *LoginService) verifyAccountStillValid(email string, expectedAccountId string) (bool, error) {
	accountGrpc, err := l.getAccountByEmail(email)
	if err != nil {
		return false, err
	}

	if account := accountGrpc.GetAccount(); account != nil && account.AccountId != "" {
		return account.AccountId == expectedAccountId, nil
	}

	return false, nil
}

func (l *LoginService) createOrUpdateSession(email string, accountId *string, code string) (*entity.LoginSession, error) {
	session, err := l.loginSessionRepository.FindByEmail(email)
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
			CreateAt:    time.Now(),
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

	if err := l.loginSessionRepository.Save(session); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	debugSession := *session
	debugSession.Code = originalCode
	l.logger.Debug(debugSession)

	return session, nil
}

func (l *LoginService) Login(email string) (*LoginAnswer, error) {
	l.cleanupExpiredSessions()

	errs := commonEntity.NewValidationError()
	l.validator.Var("email", email, "required,email", errs)
	if len(errs.FieldErrors) > 0 {
		return nil, errs
	}

	accountGrpc, err := l.getAccountByEmail(email)
	if err != nil {
		return nil, err
	}

	var accountId *string
	if account := accountGrpc.GetAccount(); account != nil && account.AccountId != "" {
		accountId = &account.AccountId
	}

	var session *entity.LoginSession

	if accountId != nil {
		plaintextCode, err := generator.Reggen(l.CodeGenConfig.CodePattern, l.CodeGenConfig.CodeMaxLength)
		if err != nil {
			l.logger.Error(err)
			return nil, commonEntity.NewInternalError()
		}

		session, err = l.createOrUpdateSession(email, accountId, plaintextCode)
		if err != nil {
			l.logger.Error(fmt.Errorf("failed to create or update login session: %w", err))
			return nil, commonEntity.NewInternalError()
		}
	} else {
		session, err = l.createOrUpdateSession(email, nil, "")
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

func (l *LoginService) validateConfirmLoginData(email string, code string) (*string, error) {
	session, err := l.loginSessionRepository.FindByEmail(email)
	if err != nil {
		l.logger.Error(err)
		return nil, commonEntity.NewInternalError()
	}
	if session == nil {
		return nil, entity.NewInvalidCodeError()
	}
	if session.CodeExpires.Before(time.Now()) {
		return nil, entity.NewCodeExpiredError()
	}

	if code == "" || !hash.VerifyCode(session.Code, code) {
		return nil, entity.NewInvalidCodeError()
	}

	return session.AccountId, nil
}

func (l *LoginService) ConfirmLogin(email string, code string) (string, error) {
	errs := commonEntity.NewValidationError()
	l.validator.Var("email", email, "required,email", errs)
	l.validator.Var("code", code, "required", errs)
	if len(errs.FieldErrors) > 0 {
		return "", errs
	}

	session, err := l.loginSessionRepository.FindByEmail(email)
	if err != nil {
		l.logger.Error(err)
		return "", commonEntity.NewInternalError()
	}
	if session == nil {
		return "", entity.NewInvalidCodeError()
	}
	if session.CodeExpires.Before(time.Now()) {
		return "", entity.NewCodeExpiredError()
	}
	if code == "" || !hash.VerifyCode(session.Code, code) {
		return "", entity.NewInvalidCodeError()
	}

	accountId := session.AccountId

	if accountId == nil {
		return "", entity.NewInvalidCodeError()
	}

	accountStillValid, _ := l.verifyAccountStillValid(email, *accountId)

	if err := l.loginSessionRepository.DeleteByEmail(context.Background(), email); err != nil {
		l.logger.Error(fmt.Errorf("failed to delete session after account switch: %w", err))
		return "", commonEntity.NewInternalError()
	}

	if !accountStillValid {
		l.logger.Info(fmt.Sprintf("Account switch detected for email %s", email))
		return "", entity.NewInvalidCodeError()
	}

	return *accountId, nil
}
