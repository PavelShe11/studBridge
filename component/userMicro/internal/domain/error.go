package domain

import (
	"github.com/PavelShe11/studbridge/common/domain"
)

var (
	ValidationError = &domain.BaseValidationError{
		BaseError:   domain.BaseError{Code: "validationError"},
		FieldErrors: make([]domain.FieldError, 0),
	}
)
