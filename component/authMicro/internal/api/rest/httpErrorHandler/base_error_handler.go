package httpErrorHandler

import (
	"errors"
	"github.com/PavelShe11/studbridge/auth/internal/domain"
	commondomain "github.com/PavelShe11/studbridge/common/domain"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"net/http"

	"github.com/labstack/echo/v4"
)

type baseErrorhandler struct {
	translator *translator.Translator
	log        logger.Logger
}

func NewBaseErrorHandler(translator *translator.Translator, log logger.Logger) DomainErrorHandler {
	return &baseErrorhandler{
		translator: translator,
		log:        log,
	}
}

func (h *baseErrorhandler) handle(err error, c echo.Context) bool {
	var domainErr commondomain.AbstractError
	ok := errors.As(err, &domainErr)
	if !ok {
		return false
	}

	statusCode, err := getStatusCodeForBaseError(domainErr.GetCode())
	if err != nil {
		statusCode = http.StatusInternalServerError
		domainErr = commondomain.InternalError
	}

	lang := GetLangFromHeader(c)

	h.translator.TranslateError(domainErr, lang)

	if err := c.JSON(statusCode, domainErr); err != nil {
		h.log.Error("Failed to send error response", "error", err)
	}

	return true
}

func getStatusCodeForBaseError(base string) (int, error) {
	switch base {
	case commondomain.InternalError.Name:
		return http.StatusInternalServerError, nil
	case domain.InvalidCode.Name, domain.CodeExpired.Name:
		return http.StatusBadRequest, nil
	default:
		return 0, errors.New("no mapping was added to the http code error for the error")
	}
}
