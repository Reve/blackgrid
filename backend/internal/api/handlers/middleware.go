package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"blackgrid/internal/metrics"
	"blackgrid/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// StructuredLogger returns a middleware that logs requests using slog.
func StructuredLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			status := res.Status
			latency := time.Since(start)

			// Record metrics
			path := c.Path() // use template path to avoid high cardinality
			if path == "" {
				path = req.URL.Path
			}
			metrics.HttpRequestsTotal.WithLabelValues(req.Method, path, strconv.Itoa(status)).Inc()
			metrics.HttpRequestDuration.WithLabelValues(req.Method, path).Observe(latency.Seconds())

			user, _ := c.Get("user").(*service.User)
			userID := ""
			if user != nil {
				userID = uuidStr(user.ID)
			}

			requestID := res.Header().Get(echo.HeaderXRequestID)

			fields := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", status),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.String("remote_ip", c.RealIP()),
			}

			if userID != "" {
				fields = append(fields, slog.String("user_id", userID))
			}

			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(req.Context(), level, "http request", fields...)

			return nil
		}
	}
}

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
		i.mu.Unlock()
	}

	return limiter
}

// RateLimitMiddleware creates a middleware that limits requests per IP.
func RateLimitMiddleware(r rate.Limit, b int, code string, message string) echo.MiddlewareFunc {
	limiter := NewIPRateLimiter(r, b)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !limiter.GetLimiter(ip).Allow() {
				return Error(c, ErrCodeRateLimited, message, nil)
			}
			return next(c)
		}
	}
}

// UserRateLimiter limits requests per user.
type UserRateLimiter struct {
	users map[string]*rate.Limiter
	mu    *sync.RWMutex
	r     rate.Limit
	b     int
}

func NewUserRateLimiter(r rate.Limit, b int) *UserRateLimiter {
	return &UserRateLimiter{
		users: make(map[string]*rate.Limiter),
		mu:    &sync.RWMutex{},
		r:     r,
		b:     b,
	}
}

func (i *UserRateLimiter) GetLimiter(userID string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.users[userID]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		limiter = rate.NewLimiter(i.r, i.b)
		i.users[userID] = limiter
		i.mu.Unlock()
	}

	return limiter
}

func UserRateLimitMiddleware(r rate.Limit, b int, code string, message string) echo.MiddlewareFunc {
	limiter := NewUserRateLimiter(r, b)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, _ := c.Get("user").(*service.User)
			if user == nil {
				return next(c)
			}
			userID := uuidStr(user.ID)
			if !limiter.GetLimiter(userID).Allow() {
				return Error(c, ErrCodeRateLimited, message, nil)
			}
			return next(c)
		}
	}
}

// SecurityHeaders adds standard security headers.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			res.Header().Set("X-Content-Type-Options", "nosniff")
			res.Header().Set("X-Frame-Options", "DENY")
			res.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Content-Security-Policy is tricky without knowing the environment,
			// but we can add a basic one.
			// res.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none';")
			return next(c)
		}
	}
}
