package accountGrpcService

import (
	"context"
	commondomain "github.com/PavelShe11/studbridge/common/domain"
	"github.com/PavelShe11/studbridge/common/translator"
	"github.com/PavelShe11/studbridge/user/internal/api/grpc"
	"github.com/PavelShe11/studbridge/user/internal/domain"
	"github.com/PavelShe11/studbridge/user/internal/service"

	grpc2 "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type accountGrpcService struct {
	grpc.UnimplementedAccountServiceServer
	accountService service.AccountService
	translator     *translator.Translator
}

func Register(server *grpc2.Server, accountService service.AccountService, trans *translator.Translator) {
	grpc.RegisterAccountServiceServer(server, &accountGrpcService{
		accountService: accountService,
		translator:     trans,
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
	return v.GetStringValue()
}

func (a accountGrpcService) CreateAccount(ctx context.Context, request *grpc.CreateAccountRequest) (*grpc.CreateAccountResponse, error) {
	lang := getLangFromContext(ctx)

	err := a.accountService.CreateAccount(domain.Account{
		FirstName: valueToString(request.UserData, "firstName"),
		LastName:  valueToString(request.UserData, "lastName"),
		Email:     valueToString(request.UserData, "email"),
	})

	// Translate errors before sending via gRPC
	if err != nil {
		a.translator.TranslateError(err, lang)
	}

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

func (a accountGrpcService) accountMapToGetAccountResponse(account *domain.Account, err error) (*grpc.GetAccountResponse, error) {
	if err != nil {
		return &grpc.GetAccountResponse{
			Result: &grpc.GetAccountResponse_Error{
				Error: mapToGrpcError(err),
			},
		}, nil
	}

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

func (a accountGrpcService) ValidateAccountData(ctx context.Context, request *grpc.ValidateAccountRequest) (*grpc.ValidateAccountResponse, error) {
	lang := getLangFromContext(ctx)

	err := a.accountService.ValidateAccountData(domain.Account{
		FirstName: valueToString(request.UserData, "firstName"),
		LastName:  valueToString(request.UserData, "lastName"),
		Email:     valueToString(request.UserData, "email"),
	})

	// Translate errors before sending via gRPC
	if err != nil {
		a.translator.TranslateError(err, lang)
	}

	return &grpc.ValidateAccountResponse{
		Error: mapToGrpcError(err),
	}, nil
}

func mapToGrpcError(e error) *grpc.Error {
	if e == nil {
		return nil
	}

	errs := make([]*grpc.Error_FieldError, 0)

	// Try to type assert to BaseValidationError first (has field errors)
	if validErr, ok := e.(*commondomain.BaseValidationError); ok {
		for _, err := range validErr.FieldErrors {
			errs = append(errs, &grpc.Error_FieldError{
				Name:    err.NameField,
				Message: err.Message,
			})
		}
		return &grpc.Error{
			Name:           validErr.Name,
			DetailedErrors: errs,
		}
	}

	// Try to type assert to BaseError (no field errors)
	if baseErr, ok := e.(*commondomain.BaseError); ok {
		return &grpc.Error{
			Name:           baseErr.Name,
			DetailedErrors: errs,
		}
	}

	// Fallback for any other error
	return &grpc.Error{
		Name:           e.Error(),
		DetailedErrors: errs,
	}
}

func getLangFromContext(ctx context.Context) string {
	lang := "en" // Default language
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if langs := md.Get("lang"); len(langs) > 0 {
			lang = langs[0]
		}
	}
	return lang
}
