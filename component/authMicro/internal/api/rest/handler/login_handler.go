package handler

import (
	"github.com/PavelShe11/studbridge/auth/internal/service"
	"github.com/PavelShe11/studbridge/common/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type Login struct {
	logger       logger.Logger
	loginService service.LoginService
}

func NewLoginHandler(logger logger.Logger, loginService service.LoginService) *Login {
	return &Login{
		logger:       logger,
		loginService: loginService,
	}
}

func (h *Login) SendLoginCode(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, req)
}

func (h *Login) ConfirmEmail(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, req)
}
