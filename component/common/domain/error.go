package domain

import (
	"fmt"
	"strings"
)

// TranslatableError interface for errors that can be translated
type TranslatableError interface {
	error
	Translate(translate func(msgID string, params map[string]interface{}) string)
}

type AbstractError interface {
	error
	TranslatableError
	GetCode() string
}

type BaseError struct {
	Code string `json:"-"`
	Name string `json:"name"`
}

type BaseValidationError struct {
	BaseError
	FieldErrors []FieldError `json:"fieldErrors"`
}

type FieldError struct {
	NameField string            `json:"nameField"`
	Message   string            `json:"message"`
	Params    map[string]string `json:"-"` // Parameters for validation errors (not serialized to JSON)
}

var (
	InternalError = &BaseError{Code: "internalError", Name: "internalError"}
)

// BaseError implements the error interface
func (e *BaseError) Error() string {
	return e.Name
}

func (e *BaseError) Translate(translate func(msgID string, params map[string]interface{}) string) {
	e.Name = translate(e.Name, nil)
}

func (e *BaseError) GetCode() string {
	return e.Code
}

func (e *BaseValidationError) Error() string {
	if len(e.FieldErrors) == 0 {
		return e.BaseError.Error()
	}

	var fieldMessages []string
	for _, fe := range e.FieldErrors {
		fieldMessages = append(fieldMessages, fmt.Sprintf("%s: %s", fe.NameField, fe.Message))
	}

	return fmt.Sprintf("%s [%s]", e.Name, strings.Join(fieldMessages, ", "))
}

func (e *BaseValidationError) Translate(translate func(msgID string, params map[string]interface{}) string) {
	e.Name = translate(e.Name, nil)

	for i := range e.FieldErrors {
		params := make(map[string]interface{})
		for k, v := range e.FieldErrors[i].Params {
			params[k] = v
		}
		e.FieldErrors[i].Message = translate(e.FieldErrors[i].Message, params)
	}
}

func (e *BaseValidationError) GetCode() string {
	return e.Code
}
