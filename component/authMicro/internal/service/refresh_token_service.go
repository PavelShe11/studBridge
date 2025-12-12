package service

type RefreshTokenService struct{}

func NewRefreshTokenService() RefreshTokenService {
	return RefreshTokenService{}
}

func (r *RefreshTokenService) RefreshTokens(refreshToken string) {

}
