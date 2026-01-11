package error // Package error common errors

import commonEntity "github.com/PavelShe11/studbridge/common/entity"

// NewInvalidCodeError creates a new instance of InvalidCode error
func NewInvalidCodeError() *commonEntity.BaseValidationError {
	return &commonEntity.BaseValidationError{
		BaseError: commonEntity.BaseError{Code: "invalidCode"},
		FieldErrors: []commonEntity.FieldError{{
			NameField: "code",
			Message:   "invalidCode",
			Params:    nil,
		}},
	}
}

// NewCodeExpiredError creates a new instance of CodeExpired error
func NewCodeExpiredError() *commonEntity.BaseValidationError {
	return &commonEntity.BaseValidationError{
		BaseError: commonEntity.BaseError{Code: "codeExpired"},
		FieldErrors: []commonEntity.FieldError{{
			NameField: "code",
			Message:   "codeExpired",
			Params:    nil,
		}},
	}
}
