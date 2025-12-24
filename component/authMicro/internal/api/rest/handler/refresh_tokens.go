package handler

import (
	"github.com/PavelShe11/studbridge/common/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type RefreshToken struct {
	logger logger.Logger
}

func NewRefreshTokenHandler(logger logger.Logger) *RefreshToken {
	return &RefreshToken{
		logger: logger,
	}
}

func (h *RefreshToken) RefreshToken(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, req)
}
