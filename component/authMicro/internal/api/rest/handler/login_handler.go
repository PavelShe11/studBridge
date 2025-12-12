package handler

import (
	"authMicro/internal/service"
	"authMicro/utlis/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type Login struct {
	logger logger.Logger
}

func NewLoginHandler(logger logger.Logger, service service.LoginService) *Login {
	return &Login{
		logger: logger,
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
