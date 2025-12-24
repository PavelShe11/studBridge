package domain

import (
	"github.com/PavelShe11/studbridge/auth/internal/api/grpcService"
	"github.com/PavelShe11/studbridge/common/domain"
)

var (
	InvalidCode = &domain.BaseValidationError{
		BaseError: domain.BaseError{Code: "invalidCode"},
		FieldErrors: []domain.FieldError{{
			NameField: "code",
			Message:   "invalidCode",
			Params:    nil,
		}},
	}
	CodeExpired = &domain.BaseValidationError{
		BaseError: domain.BaseError{Code: "codeExpired"},
		FieldErrors: []domain.FieldError{{
			NameField: "code",
			Message:   "codeExpired",
			Params:    nil,
		}},
	}
)

func GrpcErrorMapToError(errs *grpcService.Error) *domain.BaseValidationError {
	result := domain.BaseValidationError{
		BaseError:   domain.BaseError{},
		FieldErrors: make([]domain.FieldError, 0),
	}
	for _, err := range errs.DetailedErrors {
		result.FieldErrors = append(result.FieldErrors, domain.FieldError{
			NameField: err.Name,
			Message:   err.Message,
		})
	}
	return &result
}
