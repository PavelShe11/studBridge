package rest

import (
	"authMicro/internal/api/rest/handler"
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type Router struct {
	e *echo.Echo
}

func NewRouter(
	regHandler *handler.Register,
	loginHandler *handler.Login,
	refreshTokenHandler *handler.RefreshToken,
) *Router {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	v1 := e.Group("/auth/v1")
	v1.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	v1.GET("/swagger/*", echoSwagger.EchoWrapHandler(echoSwagger.URL("/swagger/doc.json")))

	login := v1.Group("/login")
	login.POST("/sendCodeEmail", loginHandler.SendLoginCode)
	login.POST("/confirmEmail", loginHandler.ConfirmEmail)

	registration := v1.Group("/registration")
	registration.POST("", regHandler.SendRegistrationCode)
	registration.POST("/confirmEmail", regHandler.RegistrationConfirmEmail)

	v1.POST("/refreshToken", refreshTokenHandler.RefreshToken)

	return &Router{
		e: e,
	}
}

func (r *Router) Start(address string) error {
	return r.e.Start(address)
}

func (r *Router) Shutdown(ctx context.Context) error {
	return r.e.Shutdown(ctx)
}
