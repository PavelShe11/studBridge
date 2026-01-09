package tokenGenerator

import (
	"fmt"

	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type JwtTokenAdapter struct {
	secret        []byte
	signingMethod jwt.SigningMethod
}

var _ TokenGenerator = (*JwtTokenAdapter)(nil)

func NewJwtTokenAdapter(jwtConfig config.JWTConfig) TokenGenerator {
	return &JwtTokenAdapter{
		secret:        []byte(jwtConfig.Secret),
		signingMethod: jwt.SigningMethodHS256,
	}
}

func (a *JwtTokenAdapter) GenerateToken(claims TokenClaims) (string, error) {
	jwtClaims := jwt.MapClaims{
		"sub": claims.Subject,
		"iat": claims.IssuedAt.Unix(),
		"nbf": claims.NotBefore.Unix(),
		"exp": claims.ExpiresAt.Unix(),
	}

	// Добавляем Extra claims
	if claims.Extra != nil {
		for key, value := range claims.Extra {
			jwtClaims[key] = value
		}
	}

	token := jwt.NewWithClaims(a.signingMethod, jwtClaims)
	tokenString, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (a *JwtTokenAdapter) ParseToken(tokenString string) (*ParsedToken, error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.secret, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("sub claim not found or invalid")
	}

	claimsMap := make(map[string]interface{})
	for key, value := range claims {
		claimsMap[key] = value
	}

	return &ParsedToken{
		Subject: sub,
		Claims:  claimsMap,
		Valid:   token.Valid,
	}, nil
}
