package service

import (
	"context"
	"fmt"
	"time"

	"github.com/PavelShe11/studbridge/auth/internal/config"
	"github.com/PavelShe11/studbridge/auth/internal/entity"
	"github.com/PavelShe11/studbridge/auth/internal/repository"
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/golang-jwt/jwt/v5"
)

/**
TODO: Поработать с context
*/

type Tokens struct {
	AccessToken         string `json:"accessToken"`
	AccessTokenExpires  int64  `json:"accessTokenExpires"`
	RefreshToken        string `json:"refreshToken"`
	RefreshTokenExpires int64  `json:"refreshTokenExpires"`
}

type TokenService struct {
	refreshTokenSessionRepo *repository.RefreshTokenSessionRepository
	accountServiceClient    grpcApi.AccountServiceClient
	jwtConfig               config.JWTConfig
	logger                  logger.Logger
}

func NewTokenService(
	refreshTokenSessionRepo *repository.RefreshTokenSessionRepository,
	accountServiceClient grpcApi.AccountServiceClient,
	logger logger.Logger,
	jwtConfig config.JWTConfig,
) *TokenService {
	return &TokenService{
		jwtConfig:               jwtConfig,
		refreshTokenSessionRepo: refreshTokenSessionRepo,
		accountServiceClient:    accountServiceClient,
		logger:                  logger,
	}
}

func (s *TokenService) CreateTokens(accountId string) (*Tokens, error) {
	s.cleanupExpiredSessions()
	payloadResp, err := s.accountServiceClient.GetAccessTokenPayload(
		context.Background(),
		&grpcApi.GetAccessTokenPayloadRequest{AccountId: accountId},
	)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to get token payload: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	if grpcError := payloadResp.GetError(); grpcError != nil {
		s.logger.Error("user service returned error for token payload")
		return nil, entity.GrpcErrorMapToError(grpcError)
	}

	claimsResult := payloadResp.GetClaims()
	if claimsResult != nil && claimsResult.GetValues()["sub"] != nil {
		accountId = claimsResult.GetValues()["sub"].GetStringValue()
	}

	refreshExpiry := time.Now().Add(s.jwtConfig.RefreshTokenExpiration)

	claimsMap := jwt.MapClaims{
		"sub": accountId,
		"exp": refreshExpiry.Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsMap)

	if claimsResult != nil {
		for key, value := range claimsResult.Values {
			claimsMap[key] = value.AsInterface()
		}
	}

	accessExpiry := time.Now().Add(s.jwtConfig.AccessTokenExpiration)

	claimsMap["exp"] = accessExpiry.Unix()
	claimsMap["iat"] = time.Now().Unix()
	claimsMap["nbf"] = time.Now().Unix()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsMap)

	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to sign refresh token: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	accessTokenString, err := accessToken.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to sign access token: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	session := &entity.RefreshTokenSession{
		AccountID:    accountId,
		RefreshToken: refreshTokenString,
		ExpiresAt:    refreshExpiry,
	}

	if err := s.refreshTokenSessionRepo.Save(session); err != nil {
		s.logger.Error(fmt.Errorf("failed to save refresh token session: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	return &Tokens{
		AccessToken:         accessTokenString,
		AccessTokenExpires:  accessExpiry.Unix(),
		RefreshToken:        refreshTokenString,
		RefreshTokenExpires: accessExpiry.Unix(),
	}, nil
}

func (s *TokenService) RefreshTokens(refreshTokenString string) (*Tokens, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		refreshTokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.jwtConfig.Secret), nil
		},
	)

	if err != nil || !token.Valid {
		s.logger.Debug(err)
		return nil, entity.NewInvalidRefreshTokenError()
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		s.logger.Error("sub not found in refresh token")
		return nil, entity.NewInvalidRefreshTokenError()
	}

	session, err := s.refreshTokenSessionRepo.FindByToken(refreshTokenString)
	if err != nil {
		s.logger.Error(err)
		return nil, commonEntity.NewInternalError()
	}
	if session == nil {
		return nil, entity.NewInvalidRefreshTokenError()
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = s.refreshTokenSessionRepo.DeleteByToken(refreshTokenString)
		return nil, entity.NewRefreshTokenExpiredError()
	}

	if err := s.refreshTokenSessionRepo.DeleteByToken(refreshTokenString); err != nil {
		s.logger.Error(err)
	}

	return s.CreateTokens(sub)
}

func (s *TokenService) cleanupExpiredSessions() {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelCleanup()
	if err := s.refreshTokenSessionRepo.CleanExpired(cleanupCtx); err != nil {
		s.logger.Error(fmt.Errorf("error cleaning expired refresh token sessions: %w", err))
	}
}
