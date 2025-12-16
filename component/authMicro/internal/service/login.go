package service

import "authMicro/internal/api/grpcService"

type LoginService struct {
	accountService grpcService.AccountServiceClient
}

func NewLoginService(accountService grpcService.AccountServiceClient) LoginService {
	return LoginService{
		accountService: accountService,
	}
}

func (l *LoginService) Login(email string) {

}

func (l *LoginService) ConfirmLoginEmail(email string, code string, id string, userAgent string) {

}
