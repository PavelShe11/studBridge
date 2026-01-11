package handler

import (
	"errors"

	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/common/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type RefreshToken struct {
	logger       logger.Logger
	tokenService *service.TokenService
}

func NewRefreshTokenHandler(logger logger.Logger, tokenService *service.TokenService) *RefreshToken {
	return &RefreshToken{
		logger:       logger,
		tokenService: tokenService,
	}
}

func (h *RefreshToken) RefreshToken(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	refreshToken, ok := req["refreshToken"].(string)
	if !ok || refreshToken == "" {
		return c.NoContent(http.StatusUnauthorized)
	}

	tokens, err := h.tokenService.RefreshTokens(c.Request().Context(), refreshToken)
	if err != nil {
		if errors.Is(err, service.InvalidRefreshTokenError) ||
			errors.Is(err, service.RefreshTokenExpiredError) ||
			errors.Is(err, service.UnauthorizedRefreshTokenError) {

			return c.NoContent(http.StatusUnauthorized)
		}

		return err
	}

	return c.JSON(http.StatusOK, models.NewTokensResponse(tokens))
}
