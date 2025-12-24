package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/PavelShe11/studbridge/common/logger"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RequestLogger(log logger.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogHeaders:  []string{"Content-Type", "User-Agent"},
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				log.Infof("%s %s %d %s %s", v.Method, v.URI, v.Status, v.Latency, v.RemoteIP)
			} else {
				log.Errorf("%s %s %d %s %s error=%v", v.Method, v.URI, v.Status, v.Latency, v.RemoteIP, v.Error)
			}
			return nil
		},
	})
}

func RequestBodyLogger(log logger.Logger) echo.MiddlewareFunc {
	return middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		maxSize := 1024
		if len(reqBody) > 0 {
			reqStr := formatBodyForLog(reqBody, maxSize)
			log.Debugf("Request Body:\n%s", reqStr)
		}
		if len(resBody) > 0 {
			resStr := formatBodyForLog(resBody, maxSize)
			log.Debugf("Response Body:\n%s", resStr)
		}
	})
}

// formatBodyForLog форматирует body для логирования.
// Если это JSON - форматирует с отступами, если нет - возвращает как есть
func formatBodyForLog(body []byte, maxSize int) string {
	if len(body) == 0 {
		return ""
	}
	var js interface{}
	if err := json.Unmarshal(body, &js); err != nil {
		// Не JSON - вернуть как есть (убрать trailing newlines)
		str := strings.TrimRight(string(body), "\n\r")
		if len(str) > maxSize {
			return str[:maxSize] + "..."
		}
		return str
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		str := strings.TrimRight(string(body), "\n\r")
		if len(str) > maxSize {
			return str[:maxSize] + "..."
		}
		return str
	}
	formatted := strings.TrimRight(buf.String(), "\n\r")
	if len(formatted) > maxSize {
		return formatted[:maxSize] + "..."
	}
	return formatted
}
