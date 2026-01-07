package usecase

import (
	"context"

	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

type AuthenticateUser struct {
	loginService *service.LoginService
	tokenService *service.TokenService
	trManager    *manager.Manager
}

func NewAuthenticateUser(
	loginService *service.LoginService,
	tokenService *service.TokenService,
	trManager *manager.Manager,
) *AuthenticateUser {
	return &AuthenticateUser{
		loginService: loginService,
		tokenService: tokenService,
		trManager:    trManager,
	}
}

func (a *AuthenticateUser) Execute(ctx context.Context, email, code string) (*service.Tokens, error) {
	var tokens *service.Tokens
	var err error
	err = a.trManager.Do(ctx, func(ctx context.Context) error {
		var accountId string
		accountId, err = a.loginService.ConfirmLogin(ctx, email, code)
		if err != nil {
			return err
		}

		tokens, err = a.tokenService.CreateTokens(ctx, accountId)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return tokens, nil
}
