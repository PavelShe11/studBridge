package service

import (
	"authMicro/internal/api/grpcService"
	"authMicro/internal/config"
	"authMicro/internal/domain"
	"authMicro/internal/repository"
	"authMicro/utlis/converter"
	"authMicro/utlis/generator"
	"authMicro/utlis/logger"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"google.golang.org/grpc/status"
)

/*
TODO: Добавить хэширвоание кодов
TODO: Изменить отображение ошибки сейчас поля, к которым относится ошибка носят названия как в структуре а нужно чтобы как в body запроса
TODO: Подумать о добавлении транзакций
*/

type RegisterAnswer struct {
	CodeExpires int64  `json:"codeExpires"`
	CodePattern string `json:"codePattern"`
}

type RegistrationService struct {
	registrationSessionRepository repository.RegistrationSessionRepository
	accountServiceClient          grpcService.AccountServiceClient
	logger                        logger.Logger
	CodeGenConfig                 *config.CodeGenConfig
}

func NewRegistrationService(
	registrationSessionRepository repository.RegistrationSessionRepository,
	accountServiceClient grpcService.AccountServiceClient,
	logger logger.Logger,
	codeGenConfig *config.CodeGenConfig,
) RegistrationService {
	return RegistrationService{
		registrationSessionRepository: registrationSessionRepository,
		accountServiceClient:          accountServiceClient,
		logger:                        logger,
		CodeGenConfig:                 codeGenConfig,
	}
}

func (r *RegistrationService) validateRegistrationData(userData map[string]any) (map[string]*structpb.Value, *domain.Error) {
	grpcMap, err := converter.ConvertToGrpcMap(userData)
	if err != nil {
		r.logger.Error(err)
		return nil, &domain.Error{Name: "internalError"}
	}

	ctxV, cancelV := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelV()
	validationResponse, err := r.accountServiceClient.ValidateAccountData(
		ctxV,
		&grpcService.ValidateAccountRequest{UserData: grpcMap},
	)
	if err != nil {
		st, _ := status.FromError(err)
		r.logger.Error(fmt.Errorf("ValidateAccountData error: %v, grpc status: %v", err, st))
		return nil, &domain.Error{Name: "internalError"}
	}
	if validationResponse.Error != nil {
		return nil, domain.GrpcErrorMapToError(validationResponse.Error)
	}

	return grpcMap, nil
}

func (r *RegistrationService) getAccountByEmail(email string) (*grpcService.GetAccountResponse, *domain.Error) {
	ctxG, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	accountGrpc, err := r.accountServiceClient.GetAccountByEmail(
		ctxG,
		&grpcService.GetAccountByEmailRequest{Email: email},
	)

	if err != nil {
		st, _ := status.FromError(err)
		r.logger.Error(fmt.Errorf("GetAccountByEmail error: %v, grpc status: %v", err, st))
		return nil, &domain.Error{Name: "internalError"}
	}
	return accountGrpc, nil
}

func (r *RegistrationService) cleanupExpiredSessions() {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCleanup()
	if err := r.registrationSessionRepository.CleanExpired(cleanupCtx); err != nil {
		r.logger.Error(fmt.Errorf("error cleaning expired registration sessions: %w", err))
	}
}

func (r *RegistrationService) Register(userData map[string]any) (*RegisterAnswer, *domain.Error) {
	r.cleanupExpiredSessions()

	if _, domainErr := r.validateRegistrationData(userData); domainErr != nil {
		return nil, domainErr
	}

	email, ok := userData["email"].(string)
	if !ok {
		r.logger.Error(errors.New("email not found in response"))
		return nil, &domain.Error{Name: "internalError"}
	}

	accountGrpc, domainErr := r.getAccountByEmail(email)
	if domainErr != nil {
		return nil, domainErr
	}

	var session *domain.RegistrationSession
	var err error

	if account, ok := accountGrpc.Result.(*grpcService.GetAccountResponse_Account); ok && account != nil {
		session, err = r.createOrUpdateSession(email, "")
		if err != nil {
			r.logger.Error(err)
			return nil, &domain.Error{Name: "internalError"}
		}
	} else {
		var code string
		code, err = generator.Reggen(r.CodeGenConfig.CodePattern, r.CodeGenConfig.CodeMaxLength)
		if err != nil {
			r.logger.Error(err)
			return nil, &domain.Error{Name: "internalError"}
		}
		session, err = r.createOrUpdateSession(email, code)
		if err != nil {
			r.logger.Error(err)
			return nil, &domain.Error{Name: "internalError"}
		}
	}

	r.logger.Debug(session)

	return &RegisterAnswer{
		CodeExpires: session.CodeExpires.Unix(),
		CodePattern: r.CodeGenConfig.CodePattern,
	}, nil
}

func (r *RegistrationService) createOrUpdateSession(email string, newCode string) (*domain.RegistrationSession, error) {
	session, err := r.registrationSessionRepository.FindByEmail(email)
	if errors.Is(err, sql.ErrNoRows) {
		session = &domain.RegistrationSession{
			Code:        newCode,
			Email:       email,
			CodeExpires: time.Now().Add(r.CodeGenConfig.CodeTTL),
			CreateAt:    time.Now(),
		}
	} else if err != nil {
		r.logger.Error(err)
		return nil, err
	} else if session.CodeExpires.Before(time.Now()) {
		session.Code = newCode
		session.CodeExpires = time.Now().Add(r.CodeGenConfig.CodeTTL)
	}

	if err := r.registrationSessionRepository.Save(session); err != nil {
		r.logger.Error(err)
		return nil, err
	}

	return session, nil
}

func (r *RegistrationService) validateConfirmationCode(email string, userData map[string]any) *domain.Error {
	session, err := r.registrationSessionRepository.FindByEmail(email)
	if err != nil {
		r.logger.Error(err)
		return &domain.Error{Name: "internalError"}
	}
	if session.CodeExpires.Before(time.Now()) {
		return &domain.Error{
			Name: "codeExpired",
			FieldErrors: []domain.FieldError{
				{Name: "code", Message: "codeExpired"},
			},
		}
	}
	if code, ok := userData["code"].(string); (ok && code != session.Code) || code == "" {
		return &domain.Error{
			Name: "invalidCode",
			FieldErrors: []domain.FieldError{
				{Name: "code", Message: "invalidCode"},
			},
		}
	}
	return nil
}

func (r *RegistrationService) ConfirmRegistration(userData map[string]any) *domain.Error {
	grpcMap, domainErr := r.validateRegistrationData(userData)
	if domainErr != nil {
		return domainErr
	}

	email, ok := userData["email"].(string)
	if !ok {
		r.logger.Error(errors.New("email not found in userData"))
		return &domain.Error{Name: "internalError"}
	}

	if err := r.validateConfirmationCode(email, userData); err != nil {
		return err
	}

	ctxC, cancelC := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelC()
	createAccountResponse, err := r.accountServiceClient.CreateAccount(
		ctxC,
		&grpcService.CreateAccountRequest{UserData: grpcMap},
	)

	if err != nil {
		st, _ := status.FromError(err)
		r.logger.Error(fmt.Errorf("CreateAccount error: %v, grpc status: %v", err, st))
		return &domain.Error{Name: "internalError"}
	}

	if createAccountResponse.Error != nil {
		return domain.GrpcErrorMapToError(createAccountResponse.Error)
	}

	if err := r.registrationSessionRepository.DeleteByEmail(email); err != nil {
		r.logger.Error(err)
		return &domain.Error{Name: "internalError"}
	}

	r.logger.Info("Account successfully created for email=" + email)
	return nil
}
