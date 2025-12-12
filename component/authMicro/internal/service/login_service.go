package service

import "authMicro/internal/api/grpc"

type LoginService struct {
	accountService grpc.AccountServiceClient
}

func NewLoginService(accountService grpc.AccountServiceClient) LoginService {
	return LoginService{
		accountService: accountService,
	}
}

func (l *LoginService) Login(email string) {

}

func (l *LoginService) ConfirmLoginEmail(email string, code string, id string, userAgent string) {

}
