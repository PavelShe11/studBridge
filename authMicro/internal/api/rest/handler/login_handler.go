package handler

import (
	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/authMicro/internal/usecase"
	"github.com/PavelShe11/studbridge/common/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type Login struct {
	logger                  logger.Logger
	loginService            *service.LoginService
	authenticateUserUsecase *usecase.AuthenticateUser
}

func NewLoginHandler(
	logger logger.Logger,
	loginService *service.LoginService,
	authenticateUserUsecase *usecase.AuthenticateUser,
) *Login {
	return &Login{
		logger:                  logger,
		loginService:            loginService,
		authenticateUserUsecase: authenticateUserUsecase,
	}
}

func (h *Login) SendLoginCode(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		h.logger.Error(err)
		return err
	}

	email, ok := req["email"].(string)
	if !ok {
		email = ""
	}
	var answer, err = h.loginService.Login(c.Request().Context(), email)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, models.NewLoginResponse(answer))
}

func (h *Login) ConfirmEmail(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		h.logger.Error(err)
		return err
	}

	email, ok := req["email"].(string)
	if !ok {
		email = ""
	}
	code, ok := req["code"].(string)
	if !ok {
		code = ""
	}

	tokens, err := h.authenticateUserUsecase.Execute(c.Request().Context(), email, code)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, models.NewTokensResponse(tokens))
}
