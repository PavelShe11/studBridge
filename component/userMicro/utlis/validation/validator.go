package validation

import (
	"errors"
	"reflect"
	"strings"
	"userMicro/internal/domain"

	"github.com/go-playground/validator/v10"
)

// Singleton validator instance configured to use JSON field names
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Configure validator to use JSON tag names instead of struct field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Extract the JSON tag (handle format: "json:\"fieldName,omitempty\"")
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		// Fallback to struct field name if JSON tag is empty or "-"
		if name == "" || name == "-" {
			return fld.Name
		}

		return name
	})
}

func Var(nameField string, field interface{}, tag string, error *domain.Error) {
	err := validate.Var(field, tag)
	if err == nil {
		return
	}
	errorField := domain.FieldError{
		Name: nameField,
	}
	var validErr validator.ValidationErrors
	errors.As(err, &validErr)
	for i, err := range validErr {
		errorField.Message += err.Tag()
		if len(validErr) < i+1 {
			errorField.Message += ","
		}
	}
	error.FieldErrors = append(error.FieldErrors, errorField)
}

func Struct(s interface{}) []domain.FieldError {
	result := make([]domain.FieldError, 0)
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	var validErr validator.ValidationErrors
	errors.As(err, &validErr)
	for _, err := range validErr {
		result = append(result, domain.FieldError{
			Name:    err.Field(),
			Message: err.Tag(),
		})
	}
	return result
}
