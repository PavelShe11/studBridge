package validation

import (
	"errors"
	commondomain "github.com/PavelShe11/studbridge/common/domain"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator holds the validation logic (without translation)
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	validate := validator.New()

	// Register function to use json field names in validation errors
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})

	return &Validator{
		validate: validate,
	}
}

// Var validates a single variable and returns validation tag as error key
func (v *Validator) Var(nameField string, field interface{}, tag string, error *commondomain.BaseValidationError) {
	err := v.validate.Var(field, tag)
	if err == nil {
		return
	}

	var validErr validator.ValidationErrors
	errors.As(err, &validErr)

	for _, e := range validErr {
		error.FieldErrors = append(error.FieldErrors, commondomain.FieldError{
			NameField: nameField,
			Message:   e.Tag(), // Returns validation tag: "required", "email", etc.
			Params:    extractValidationParams(e),
		})
	}
}

// Struct validates a struct and returns validation tags as error keys
func (v *Validator) Struct(s interface{}) []commondomain.FieldError {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	result := make([]commondomain.FieldError, 0)
	var validErr validator.ValidationErrors
	errors.As(err, &validErr)

	for _, err := range validErr {
		result = append(result, commondomain.FieldError{
			NameField: err.Field(),
			Message:   err.Tag(), // Returns validation tag: "required", "email", "min", etc.
			Params:    extractValidationParams(err),
		})
	}
	return result
}

// extractValidationParams extracts parameters from validation errors
// For example: min=6 -> {"value": "6"}
func extractValidationParams(err validator.FieldError) map[string]string {
	params := make(map[string]string)

	switch err.Tag() {
	case "min", "max", "len", "gte", "lte", "gt", "lt", "eq", "ne":
		params["value"] = err.Param()
	case "oneof":
		params["values"] = err.Param()
	case "eqfield", "nefield", "gtfield", "ltfield":
		params["field"] = err.Param()
	}

	return params
}
