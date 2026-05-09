package handlers

import (
	"net/http"
	"strings"

	"blackgrid/internal/db"
	"blackgrid/internal/service"

	"github.com/labstack/echo/v4"
)

const (
	cookieName    = "blackgrid_session"
	ctxKeyUser    = "auth_user"
	ctxKeyRole    = "auth_role"
	ctxKeyTokenID = "auth_token_id"
)

// AuthMiddleware returns middleware that validates the session cookie or Bearer token.
// It sets the authenticated user into the Echo context.
func AuthMiddleware(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// 1. Try session cookie first.
			cookie, err := c.Cookie(cookieName)
			if err == nil && cookie.Value != "" {
				user, _, err := authSvc.ResolveSession(ctx, cookie.Value)
				if err == nil {
					c.Set(ctxKeyUser, user)
					c.Set(ctxKeyRole, user.Role)
					return next(c)
				}
			}

			// 2. Try Bearer token.
			authHeader := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				raw := strings.TrimPrefix(authHeader, "Bearer ")
				user, tok, err := authSvc.ResolveAPIToken(ctx, raw)
				if err == nil {
					c.Set(ctxKeyUser, user)
					// API token role governs, not the user's role.
					c.Set(ctxKeyRole, tok.Role)
					c.Set(ctxKeyTokenID, tok.ID)
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
		}
	}
}

// RequireRole returns middleware that checks the authenticated user's role.
func RequireRole(minRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get(ctxKeyRole).(string)
			if !roleAtLeast(role, minRole) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}

// RequireAdmin is shorthand for RequireRole(admin).
func RequireAdmin() echo.MiddlewareFunc { return RequireRole(service.RoleAdmin) }

// RequireOperator allows admin and operator.
func RequireOperator() echo.MiddlewareFunc { return RequireRole(service.RoleOperator) }

// GetAuthUser retrieves the authenticated user from Echo context.
// Returns false if not present.
func GetAuthUser(c echo.Context) (db.User, bool) {
	u, ok := c.Get(ctxKeyUser).(db.User)
	return u, ok
}

// GetAuthRole retrieves the role string from context.
func GetAuthRole(c echo.Context) string {
	r, _ := c.Get(ctxKeyRole).(string)
	return r
}

// IsReadOnly returns true if the role is viewer.
func IsReadOnly(c echo.Context) bool {
	return GetAuthRole(c) == service.RoleViewer
}

func roleAtLeast(have, need string) bool {
	rank := map[string]int{
		service.RoleViewer:   1,
		service.RoleOperator: 2,
		service.RoleAdmin:    3,
	}
	return rank[have] >= rank[need]
}

// SetAuthCookie writes the session cookie to the response.
func SetAuthCookie(c echo.Context, hash string, cfg service.AuthConfig) {
	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = hash
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Secure = cfg.CookieSecure
	cookie.Path = "/"
	c.SetCookie(cookie)
}

// ClearAuthCookie removes the session cookie.
func ClearAuthCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.MaxAge = -1
	cookie.Path = "/"
	c.SetCookie(cookie)
}
