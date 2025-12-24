package httpErrorHandler

import (
	"strings"

	"github.com/labstack/echo/v4"
)

type DomainErrorHandler interface {
	handle(err error, c echo.Context) bool
}

func NewHttpErrorHandler(domainErrorHandlers ...DomainErrorHandler) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		for _, domainErrorHandler := range domainErrorHandlers {
			if domainErrorHandler.handle(err, c) {
				return
			}
		}
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}
}

// GetLangFromHeader extracts language preferences from the Accept-Language header.
// It returns the preferred language or "en" as default.
func GetLangFromHeader(c echo.Context) string {
	acceptLanguage := c.Request().Header.Get("Accept-Language")
	if acceptLanguage == "" {
		return "en" // Default language
	}

	// Basic parsing: split by comma and take the first one.
	// In a real app, you'd use a more robust Accept-Language parser.
	langs := strings.Split(acceptLanguage, ",")
	if len(langs) > 0 {
		return strings.TrimSpace(strings.Split(langs[0], ";")[0])
	}

	return "en" // Fallback
}
