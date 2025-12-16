package domain

import (
	"authMicro/internal/api/grpcService"

	"github.com/go-playground/validator/v10"
)

type Error struct {
	Name        string       `json:"name"`
	FieldErrors []FieldError `json:"fieldErrors"`
}

type FieldError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func ValidationErrorMapToFieldErrors(errs validator.ValidationErrors) []FieldError {
	result := make([]FieldError, 0)
	for _, err := range errs {
		result = append(result, FieldError{
			Name:    err.Field(),
			Message: err.Tag(),
		})
	}
	return result
}

func GrpcErrorMapToError(errs *grpcService.Error) *Error {
	result := Error{
		Name:        errs.Name,
		FieldErrors: make([]FieldError, 0),
	}
	for _, err := range errs.DetailedErrors {
		result.FieldErrors = append(result.FieldErrors, FieldError{
			Name:    err.Name,
			Message: err.Message,
		})
	}
	return &result
}
