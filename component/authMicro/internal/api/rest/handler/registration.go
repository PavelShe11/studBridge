package handler

import (
	"github.com/PavelShe11/studbridge/auth/internal/api/rest/httpErrorHandler"
	"github.com/PavelShe11/studbridge/auth/internal/service"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator" // Added translator import
	"net/http"

	"github.com/labstack/echo/v4"
)

type Register struct {
	logger              logger.Logger
	registrationService service.RegistrationService
	translator          *translator.Translator // Added translator field
}

func NewRegisterHandler(
	logger logger.Logger,
	registrationService service.RegistrationService,
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	lang := httpErrorHandler.GetLangFromHeader(c)            // Get language from header
	answer, err := h.registrationService.Register(req, lang) // Pass language to service
	if err != nil {
		// Translate error before returning
		h.translator.TranslateError(err, lang)
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, answer)
}

func (h *Register) RegistrationConfirmEmail(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	lang := httpErrorHandler.GetLangFromHeader(c)               // Get language from header
	err := h.registrationService.ConfirmRegistration(req, lang) // Pass language to service
	if err != nil {
		// Translate error before returning
		h.translator.TranslateError(err, lang)
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.NoContent(http.StatusOK)
}
