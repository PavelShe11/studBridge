package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PavelShe11/studbridge/authMicro/internal/entity"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	"github.com/PavelShe11/studbridge/authMicro/utlis/tokenGenerator"
	commonEntity "github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"
)

type TokenService struct {
	refreshTokenSessionRepo port.RefreshTokenSessionRepository
	accountProvider         port.AccountProvider
	tokenGenerator          tokenGenerator.TokenGenerator
	logger                  logger.Logger
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
}

var (
	InvalidRefreshTokenError      = errors.New("invalidRefreshToken")
	RefreshTokenExpiredError      = errors.New("refreshTokenExpired")
	UnauthorizedRefreshTokenError = errors.New("unauthorizedRefreshToken")
)

func NewTokenService(
	refreshTokenSessionRepo port.RefreshTokenSessionRepository,
	accountProvider port.AccountProvider,
	tokenGenerator tokenGenerator.TokenGenerator,
	logger logger.Logger,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *TokenService {
	return &TokenService{
		refreshTokenSessionRepo: refreshTokenSessionRepo,
		accountProvider:         accountProvider,
		tokenGenerator:          tokenGenerator,
		logger:                  logger,
		accessTokenTTL:          accessTokenTTL,
		refreshTokenTTL:         refreshTokenTTL,
	}
}

func (s *TokenService) CreateTokens(ctx context.Context, accountId string) (*entity.Tokens, error) {
	s.cleanupExpiredSessions(ctx)

	claimsResult, err := s.accountProvider.GetAccessTokenPayload(ctx, accountId)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to get token payload: %w", err))
		return nil, err
	}

	if claimsResult != nil {
		if sub, ok := claimsResult["sub"].(string); ok && sub != "" {
			accountId = sub
		}
	}

	now := time.Now()
	refreshExpiry := now.Add(s.refreshTokenTTL)
	accessExpiry := now.Add(s.accessTokenTTL)

	refreshTokenString, accessTokenString, err := s.generateTokenPair(
		accountId,
		claimsResult,
		now,
		refreshExpiry,
		accessExpiry,
	)
	if err != nil {
		s.logger.Error(err)
		return nil, commonEntity.NewInternalError()
	}

	session := &entity.RefreshTokenSession{
		AccountID:    accountId,
		RefreshToken: refreshTokenString,
		ExpiresAt:    refreshExpiry,
	}

	if err := s.refreshTokenSessionRepo.Save(ctx, session); err != nil {
		s.logger.Error(fmt.Errorf("failed to save refresh token session: %w", err))
		return nil, commonEntity.NewInternalError()
	}

	return &entity.Tokens{
		AccessToken:         accessTokenString,
		AccessTokenExpires:  accessExpiry.Unix(),
		RefreshToken:        refreshTokenString,
		RefreshTokenExpires: accessExpiry.Unix(),
	}, nil
}

func (s *TokenService) RefreshTokens(ctx context.Context, refreshTokenString string) (*entity.Tokens, error) {
	parsedToken, err := s.tokenGenerator.ParseToken(refreshTokenString)
	if err != nil || !parsedToken.Valid {
		s.logger.Debug(err)
		return nil, InvalidRefreshTokenError
	}

	accountId := parsedToken.Subject

	session, err := s.refreshTokenSessionRepo.FindByToken(ctx, refreshTokenString)
	if err != nil {
		s.logger.Error(err)
		return nil, commonEntity.NewInternalError()
	}
	if session == nil {
		return nil, UnauthorizedRefreshTokenError
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = s.refreshTokenSessionRepo.DeleteByToken(ctx, refreshTokenString)
		return nil, RefreshTokenExpiredError
	}

	var result *entity.Tokens
	result, err = s.CreateTokens(ctx, accountId)
	if err != nil {
		return nil, err
	}

	if err := s.refreshTokenSessionRepo.DeleteByToken(ctx, refreshTokenString); err != nil {
		s.logger.Error(err)
	}

	return result, nil
}

func (s *TokenService) generateTokenPair(
	accountId string,
	extraClaims map[string]interface{},
	now time.Time,
	refreshExpiry time.Time,
	accessExpiry time.Time,
) (refreshToken string, accessToken string, err error) {

	refreshClaims := tokenGenerator.TokenClaims{
		Subject:   accountId,
		IssuedAt:  now,
		NotBefore: now,
		ExpiresAt: refreshExpiry,
		Extra:     nil,
	}

	refreshToken, err = s.tokenGenerator.GenerateToken(refreshClaims)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	accessClaims := tokenGenerator.TokenClaims{
		Subject:   accountId,
		IssuedAt:  now,
		NotBefore: now,
		ExpiresAt: accessExpiry,
		Extra:     extraClaims,
	}

	accessToken, err = s.tokenGenerator.GenerateToken(accessClaims)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return refreshToken, accessToken, nil
}

func (s *TokenService) cleanupExpiredSessions(ctx context.Context) {
	if err := s.refreshTokenSessionRepo.CleanExpired(ctx); err != nil {
		s.logger.Error(fmt.Errorf("error cleaning expired refresh token sessions: %w", err))
	}
}
