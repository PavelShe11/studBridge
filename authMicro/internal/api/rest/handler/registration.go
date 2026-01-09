package handler

import (
	"net/http"

	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/httpErrorHandler"
	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/models"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator" // Added translator import

	"github.com/labstack/echo/v4"
)

type Register struct {
	logger              logger.Logger
	registrationService *service.RegistrationService
	translator          *translator.Translator // Added translator field
}

func NewRegisterHandler(
	logger logger.Logger,
	registrationService *service.RegistrationService,
	translator *translator.Translator, // Added translator parameter
) *Register {
	return &Register{
		logger:              logger,
		registrationService: registrationService,
		translator:          translator, // Assign translator
	}
}

func (h *Register) SendRegistrationCode(c echo.Context) error {
	var req map[string]any
	if err := c.Bind(&req); err != nil {
		h.logger.Error(err)
		return err
	}

	lang := httpErrorHandler.GetLangFromHeader(c)
	answer, err := h.registrationService.Register(c.Request().Context(), req, lang)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, models.NewRegistrationResponse(answer))
}

func (h *Register) RegistrationConfirmEmail(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		h.logger.Error(err)
		return err
	}

	lang := httpErrorHandler.GetLangFromHeader(c)
	err := h.registrationService.ConfirmRegistration(c.Request().Context(), req, lang)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}
