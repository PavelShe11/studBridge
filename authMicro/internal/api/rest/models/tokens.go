package models

import "github.com/PavelShe11/studbridge/authMicro/internal/entity"

// TokensResponse represents the authentication tokens returned to clients
type TokensResponse struct {
	AccessToken         string `json:"accessToken"`
	AccessTokenExpires  int64  `json:"accessTokenExpires"`
	RefreshToken        string `json:"refreshToken"`
	RefreshTokenExpires int64  `json:"refreshTokenExpires"`
}

// NewTokensResponse creates a TokensResponse from domain entity
func NewTokensResponse(t *entity.Tokens) *TokensResponse {
	return &TokensResponse{
		AccessToken:         t.AccessToken,
		AccessTokenExpires:  t.AccessTokenExpires,
		RefreshToken:        t.RefreshToken,
		RefreshTokenExpires: t.RefreshTokenExpires,
	}
}
