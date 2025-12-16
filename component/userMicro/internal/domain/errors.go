package domain

import (
	"github.com/go-playground/validator/v10"
)

type Error struct {
	Name        string
	FieldErrors []FieldError
}

type FieldError struct {
	Name    string
	Message string
}

func ValidationErrorsMap(errs validator.ValidationErrors) []FieldError {
	var result = make([]FieldError, 0)
	for _, err := range errs {
		result = append(result, FieldError{
			Name:    err.Field(),
			Message: err.Tag(),
		})
	}
	return result
}
