package http

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewRouter creates an Echo router with all routes and middleware.
func NewRouter(h *Handler, rateLimiter *IPRateLimiter, allowOrigin string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware stack.
	e.Use(middleware.Recover())
	e.Use(RequestIDMiddleware())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{allowOrigin},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))
	e.Use(LoggingMiddleware())
	e.Use(rateLimiter.Middleware())

	// Routes.
	e.GET("/healthz", h.Healthz)
	e.POST("/api/ask", h.Ask)

	return e
}
