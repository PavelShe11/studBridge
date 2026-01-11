package grpc

import (
	"context"
	"fmt"

	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	"github.com/PavelShe11/studbridge/authMicro/utlis/converter"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type accountGrpcAdapter struct {
	grpcClient grpcApi.AccountServiceClient
	logger     logger.Logger
}

// Проверка реализации интерфейса на этапе компиляции
var _ port.AccountProvider = (*accountGrpcAdapter)(nil)

func NewAccountGrpcAdapter(
	grpcClient grpcApi.AccountServiceClient,
	logger logger.Logger,
) port.AccountProvider {
	return &accountGrpcAdapter{
		grpcClient: grpcClient,
		logger:     logger,
	}
}

func (a *accountGrpcAdapter) ValidateAccountData(
	ctx context.Context,
	userData map[string]interface{},
	lang string,
) error {
	// Конвертация map → protobuf (используем существующий converter)
	grpcMap, err := converter.ConvertToGrpcMap(userData)
	if err != nil {
		a.logger.Error(err)
		return commonEntity.NewInternalError()
	}

	// Добавляем metadata для языка
	md := metadata.Pairs("lang", lang)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// gRPC вызов
	response, err := a.grpcClient.ValidateAccountData(ctx, &grpcApi.ValidateAccountRequest{UserData: grpcMap})
	if err != nil {
		st, _ := status.FromError(err)
		a.logger.Error(fmt.Errorf("ValidateAccountData gRPC error: %v, status: %v", err, st))
		return commonEntity.NewInternalError()
	}

	// Конвертация gRPC error → domain error (используем существующую функцию из entity)
	if response.Error != nil {
		return GrpcErrorMapToError(response.Error)
	}

	return nil
}

func (a *accountGrpcAdapter) CreateAccount(ctx context.Context, userData map[string]interface{}, lang string) error {
	grpcMap, err := converter.ConvertToGrpcMap(userData)
	if err != nil {
		a.logger.Error(err)
		return commonEntity.NewInternalError()
	}

	md := metadata.Pairs("lang", lang)
	ctx = metadata.NewOutgoingContext(ctx, md)

	response, err := a.grpcClient.CreateAccount(ctx, &grpcApi.CreateAccountRequest{UserData: grpcMap})
	if err != nil {
		st, _ := status.FromError(err)
		a.logger.Error(fmt.Errorf("CreateAccount gRPC error: %v, status: %v", err, st))
		return commonEntity.NewInternalError()
	}

	if response.Error != nil {
		return GrpcErrorMapToError(response.Error)
	}

	return nil
}

func (a *accountGrpcAdapter) GetAccountByEmail(ctx context.Context, email string) (*entity.Account, error) {
	response, err := a.grpcClient.GetAccountByEmail(ctx, &grpcApi.GetAccountByEmailRequest{Email: email})
	if err != nil {
		st, _ := status.FromError(err)
		a.logger.Error(fmt.Errorf("GetAccountByEmail gRPC error: %v, status: %v", err, st))
		return nil, commonEntity.NewInternalError()
	}

	// Обработка oneof result
	if account := response.GetAccount(); account != nil {
		return &entity.Account{
			AccountId: account.AccountId,
			Email:     account.Email,
		}, nil
	}

	if grpcErr := response.GetError(); grpcErr != nil {
		return nil, GrpcErrorMapToError(grpcErr)
	}

	// Аккаунт не найден
	return nil, nil
}

func (a *accountGrpcAdapter) GetAccessTokenPayload(ctx context.Context, accountId string) (map[string]interface{}, error) {
	response, err := a.grpcClient.GetAccessTokenPayload(ctx, &grpcApi.GetAccessTokenPayloadRequest{AccountId: accountId})
	if err != nil {
		st, _ := status.FromError(err)
		a.logger.Error(fmt.Errorf("GetAccessTokenPayload gRPC error: %v, status: %v", err, st))
		return nil, commonEntity.NewInternalError()
	}

	if grpcErr := response.GetError(); grpcErr != nil {
		return nil, GrpcErrorMapToError(grpcErr)
	}

	if claims := response.GetClaims(); claims != nil {
		// Конвертация protobuf map → обычный map
		values := make(map[string]interface{})
		for key, pbValue := range claims.Values {
			values[key] = pbValue.AsInterface()
		}
		return values, nil
	}

	return nil, commonEntity.NewInternalError()
}
