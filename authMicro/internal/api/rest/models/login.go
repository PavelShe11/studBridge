package models

import "github.com/PavelShe11/studbridge/authMicro/internal/service"

// LoginResponse represents the response after initiating login
type LoginResponse struct {
	CodeExpires int64  `json:"codeExpires"`
	CodePattern string `json:"codePattern"`
}

// NewLoginResponse creates a LoginResponse from service answer
func NewLoginResponse(answer *service.LoginAnswer) *LoginResponse {
	return &LoginResponse{
		CodeExpires: answer.CodeExpires,
		CodePattern: answer.CodePattern,
	}
}
