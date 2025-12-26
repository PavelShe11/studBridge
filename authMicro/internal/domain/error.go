package domain

import (
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
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
	ValidationError = &domain.BaseValidationError{BaseError: domain.BaseError{Code: "validationError"}}
)

func GrpcErrorMapToError(grpcErr *grpcApi.Error) error {
	if grpcErr == nil {
		return nil
	}

	fieldErrors := make([]domain.FieldError, 0, len(grpcErr.DetailedErrors))
	for _, err := range grpcErr.DetailedErrors {
		fieldErrors = append(fieldErrors, domain.FieldError{
			NameField: err.Name,
			Message:   err.Message,
		})
	}

	switch grpcErr.Code {
	case grpcApi.ErrorCode_VALIDATION:
		validationError := ValidationError
		validationError.FieldErrors = fieldErrors
		return validationError
	default:
		return domain.InternalError
	}
}
