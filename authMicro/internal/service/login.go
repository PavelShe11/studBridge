package service

import (
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
)

type LoginService struct {
	accountService grpcApi.AccountServiceClient
}

func NewLoginService(accountService grpcApi.AccountServiceClient) LoginService {
	return LoginService{
		accountService: accountService,
	}
}

func (l *LoginService) Login(email string) {

}

func (l *LoginService) ConfirmLoginEmail(email string, code string, id string, userAgent string) {

}
