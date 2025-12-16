package accountGrpcService

import (
	"context"
	"userMicro/internal/api/grpc"
	"userMicro/internal/domain"
	"userMicro/internal/service"

	grpc2 "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type accountGrpcService struct {
	grpc.UnimplementedAccountServiceServer
	accountService service.AccountService
}

func Register(server *grpc2.Server, accountService service.AccountService) {
	grpc.RegisterAccountServiceServer(server, &accountGrpcService{
		accountService: accountService,
	})
}

func valueToString(m map[string]*structpb.Value, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	// use GetStringValue to be explicit
	return v.GetStringValue()
}

func (a accountGrpcService) CreateAccount(_ context.Context, request *grpc.CreateAccountRequest) (*grpc.CreateAccountResponse, error) {
	err := a.accountService.CreateAccount(domain.Account{
		FirstName: valueToString(request.UserData, "firstName"),
		LastName:  valueToString(request.UserData, "lastName"),
		Email:     valueToString(request.UserData, "email"),
	})
	return &grpc.CreateAccountResponse{
		Error: mapToGrpcError(err),
	}, nil
}

func (a accountGrpcService) GetAccountByEmail(_ context.Context, request *grpc.GetAccountByEmailRequest) (*grpc.GetAccountResponse, error) {
	return a.accountMapToGetAccountResponse(
		a.accountService.GetAccountByEmail(
			request.GetEmail(),
		),
	)
}

func (a accountGrpcService) GetAccountById(_ context.Context, request *grpc.GetAccountByIdRequest) (*grpc.GetAccountResponse, error) {
	return a.accountMapToGetAccountResponse(
		a.accountService.GetAccountById(
			request.GetAccountId(),
		),
	)
}

func (a accountGrpcService) accountMapToGetAccountResponse(account *domain.Account, err *domain.Error) (*grpc.GetAccountResponse, error) {
	if err != nil {
		return &grpc.GetAccountResponse{
			Result: &grpc.GetAccountResponse_Error{
				Error: mapToGrpcError(err),
			},
		}, nil
	}

	// account expected to be non-nil when err == nil
	if account == nil {
		return &grpc.GetAccountResponse{
			Result: &grpc.GetAccountResponse_Error{
				Error: &grpc.Error{Name: "internalError"},
			},
		}, nil
	}

	return &grpc.GetAccountResponse{
		Result: &grpc.GetAccountResponse_Account{
			Account: &grpc.GetAccountResponse_AccountWrapper{
				UserData: map[string]*structpb.Value{
					"firstName": structpb.NewStringValue(account.FirstName),
					"lastName":  structpb.NewStringValue(account.LastName),
					"email":     structpb.NewStringValue(account.Email),
				},
			},
		},
	}, nil
}

func (a accountGrpcService) ValidateAccountData(_ context.Context, request *grpc.ValidateAccountRequest) (*grpc.ValidateAccountResponse, error) {
	err := a.accountService.ValidateAccountData(domain.Account{
		FirstName: valueToString(request.UserData, "firstName"),
		LastName:  valueToString(request.UserData, "lastName"),
		Email:     valueToString(request.UserData, "email"),
	})
	return &grpc.ValidateAccountResponse{
		Error: mapToGrpcError(err),
	}, nil
}

func mapToGrpcError(e *domain.Error) *grpc.Error {
	if e == nil {
		return nil
	}
	errs := make([]*grpc.Error_FieldError, 0)
	for _, err := range e.FieldErrors {
		errs = append(errs, &grpc.Error_FieldError{
			Name:    err.Name,
			Message: err.Message,
		})
	}
	return &grpc.Error{
		Name:           e.Name,
		DetailedErrors: errs,
	}
}
