package grpc

import (
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
)

func GrpcErrorMapToError(grpcErr *grpcApi.Error) error {
	if grpcErr == nil {
		return nil
	}

	fieldErrors := make([]commonEntity.FieldError, 0, len(grpcErr.DetailedErrors))
	for _, err := range grpcErr.DetailedErrors {
		fieldErrors = append(fieldErrors, commonEntity.FieldError{
			NameField: err.Name,
			Message:   err.Message,
		})
	}

	switch grpcErr.Code {
	case grpcApi.ErrorCode_VALIDATION:
		validationError := commonEntity.NewValidationError()
		validationError.FieldErrors = fieldErrors
		return validationError
	default:
		return commonEntity.NewInternalError()
	}
}
