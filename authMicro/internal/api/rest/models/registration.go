package models

import "github.com/PavelShe11/studbridge/authMicro/internal/service"

// RegistrationResponse represents the response after initiating registration
type RegistrationResponse struct {
	CodeExpires int64  `json:"codeExpires"`
	CodePattern string `json:"codePattern"`
}

// NewRegistrationResponse creates a RegistrationResponse from service answer
func NewRegistrationResponse(answer *service.RegisterAnswer) *RegistrationResponse {
	return &RegistrationResponse{
		CodeExpires: answer.CodeExpires,
		CodePattern: answer.CodePattern,
	}
}
