package handler

import (
	models2 "github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/inbound/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/authMicro/internal/usecase"
	"github.com/PavelShe11/studbridge/common/logger"

	"net/http"

	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/inbound/rest/httpErrorHandler"
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

// SendLoginCode godoc
// @Summary      Отправить код входа
// @Description  Находит аккаунт по email и генерирует OTP-код для входа.
// @Description  Обязательное поле: email (string).
// @Tags         login
// @Accept       json
// @Produce      json
// @Param        request  body      object               true  "Email пользователя"
// @Success      200      {object}  models2.LoginResponse
// @Failure      400      {object}  entity.BaseValidationError
// @Failure      500      {object}  entity.BaseError
// @Router       /login/sendCodeEmail [post]
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
	lang := httpErrorHandler.GetLangFromHeader(c)
	var answer, err = h.loginService.Login(c.Request().Context(), email, lang)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, models2.NewLoginResponse(answer))
}

// ConfirmEmail godoc
// @Summary      Подтвердить вход
// @Description  Проверяет OTP-код и возвращает пару JWT токенов (access + refresh).
// @Description  Обязательные поля: email (string), code (string).
// @Tags         login
// @Accept       json
// @Produce      json
// @Param        request  body      object               true  "Email и OTP-код"
// @Success      200      {object}  models2.TokensResponse
// @Failure      400      {object}  entity.BaseValidationError
// @Failure      401      {object}  entity.BaseError
// @Failure      500      {object}  entity.BaseError
// @Router       /login/confirmEmail [post]
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
	return c.JSON(http.StatusOK, models2.NewTokensResponse(tokens))
}
