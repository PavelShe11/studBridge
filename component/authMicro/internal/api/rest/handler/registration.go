package handler

import (
	"authMicro/internal/service"
	"authMicro/utlis/logger"

	"net/http"

	"github.com/labstack/echo/v4"
)

type Register struct {
	logger              logger.Logger
	registrationService service.RegistrationService
}

func NewRegisterHandler(logger logger.Logger, registrationService service.RegistrationService) *Register {
	return &Register{
		logger:              logger,
		registrationService: registrationService,
	}
}

func (h *Register) SendRegistrationCode(c echo.Context) error {
	var req map[string]any
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	answer, err := h.registrationService.Register(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, answer)
}

func (h *Register) RegistrationConfirmEmail(c echo.Context) error {
	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err := h.registrationService.ConfirmRegistration(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.NoContent(http.StatusOK)
}
