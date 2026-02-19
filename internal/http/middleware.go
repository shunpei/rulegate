package http

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/shunpei/rulegate/internal/domain"
	"github.com/shunpei/rulegate/internal/logging"
	"golang.org/x/time/rate"
)

// RequestIDMiddleware injects a unique request ID into the context and response header.
func RequestIDMiddleware() echo.MiddlewareFunc {
	var counter uint64
	var mu sync.Mutex
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			mu.Lock()
			counter++
			id := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), counter)
			mu.Unlock()

			ctx := logging.WithRequestID(c.Request().Context(), id)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Response().Header().Set("X-Request-ID", id)
			return next(c)
		}
	}
}

// LoggingMiddleware logs request method, path, status, and duration.
func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			slog.InfoContext(c.Request().Context(), "request",
				"request_id", logging.RequestID(c.Request().Context()),
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return err
		}
	}
}

// IPRateLimiter implements per-IP token bucket rate limiting.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = lim
	}
	return lim
}

// Middleware returns an Echo middleware that enforces rate limiting.
func (l *IPRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := clientIP(c)
			if !l.getLimiter(ip).Allow() {
				return c.JSON(429, domain.ErrorResponse{
					Error: "rate limit exceeded",
					Code:  string(domain.ErrCatRateLimit),
				})
			}
			return next(c)
		}
	}
}

func clientIP(c echo.Context) string {
	// X-Forwarded-For from Cloud Run / load balancers.
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	return c.RealIP()
}
