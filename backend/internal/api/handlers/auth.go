package handlers

import (
	"errors"
	"net/http"
	"time"

	"blackgrid/internal/db"
	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)


// AuthHandler handles auth, setup, user, and API-token endpoints.
type AuthHandler struct {
	authSvc  *service.AuthService
	auditSvc *service.AuditService
	cfg      service.AuthConfig
}

func NewAuthHandler(authSvc *service.AuthService, auditSvc *service.AuditService, cfg service.AuthConfig) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, auditSvc: auditSvc, cfg: cfg}
}

// ---- Setup ----

func (h *AuthHandler) SetupStatus(c echo.Context) error {
	required, err := h.authSvc.SetupRequired(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"setup_required": required})
}

type setupAdminRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (h *AuthHandler) SetupAdmin(c echo.Context) error {
	var req setupAdminRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.DisplayName == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email, display_name, and password are required")
	}
	user, err := h.authSvc.CreateFirstAdmin(c.Request().Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrSetupAlreadyDone) {
			return echo.NewHTTPError(http.StatusConflict, "setup already completed")
		}
		if errors.Is(err, service.ErrPasswordTooShort) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	}
	return c.JSON(http.StatusCreated, safeUser(user))
}

// ---- Auth ----

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	user, sessionHash, err := h.authSvc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserDisabled) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return err
	}
	SetAuthCookie(c, sessionHash, h.cfg)
	return c.JSON(http.StatusOK, echo.Map{"user": safeUser(user)})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(cookieName)
	if err == nil && cookie.Value != "" {
		_ = h.authSvc.Logout(c.Request().Context(), cookie.Value)
	}
	ClearAuthCookie(c)
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

func (h *AuthHandler) Me(c echo.Context) error {
	user, ok := GetAuthUser(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}
	return c.JSON(http.StatusOK, echo.Map{"user": safeUser(user)})
}

// ---- User management ----

func (h *AuthHandler) ListUsers(c echo.Context) error {
	users, err := h.authSvc.ListUsers(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]map[string]any, len(users))
	for i, u := range users {
		out[i] = safeUser(u)
	}
	return c.JSON(http.StatusOK, out)
}

type createUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	Enabled     *bool  `json:"enabled"`
}

func (h *AuthHandler) CreateUser(c echo.Context) error {
	actor, _ := GetAuthUser(c)
	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Role == "" {
		req.Role = service.RoleViewer
	}
	user, err := h.authSvc.CreateUser(c.Request().Context(), actor, service.CreateUserInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		Role:        req.Role,
		Enabled:     enabled,
	})
	if err != nil {
		if errors.Is(err, service.ErrPasswordTooShort) || errors.Is(err, service.ErrInvalidRole) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, service.ErrEmailTaken) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return err
	}
	return c.JSON(http.StatusCreated, safeUser(user))
}

func (h *AuthHandler) GetUser(c echo.Context) error {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	user, err := h.authSvc.GetUser(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, safeUser(user))
}

type updateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Enabled     *bool  `json:"enabled"`
	Password    string `json:"password"`
}

func (h *AuthHandler) UpdateUser(c echo.Context) error {
	actor, _ := GetAuthUser(c)
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	var req updateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	existing, err := h.authSvc.GetUser(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	role := existing.Role
	if req.Role != "" {
		role = req.Role
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	displayName := existing.DisplayName
	if req.DisplayName != "" {
		displayName = req.DisplayName
	}
	user, err := h.authSvc.UpdateUser(c.Request().Context(), actor, id, service.UpdateUserInput{
		DisplayName: displayName,
		Role:        role,
		Enabled:     enabled,
		Password:    req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrPasswordTooShort) || errors.Is(err, service.ErrInvalidRole) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, service.ErrLastAdmin) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, safeUser(user))
}

func (h *AuthHandler) DeleteUser(c echo.Context) error {
	actor, _ := GetAuthUser(c)
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	if err := h.authSvc.DeleteUser(c.Request().Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		if errors.Is(err, service.ErrLastAdmin) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ---- API Tokens ----

func (h *AuthHandler) ListAPITokens(c echo.Context) error {
	tokens, err := h.authSvc.ListAPITokens(c.Request().Context())
	if err != nil {
		return err
	}
	out := make([]map[string]any, len(tokens))
	for i, t := range tokens {
		out[i] = safeToken(t)
	}
	return c.JSON(http.StatusOK, out)
}

type createAPITokenRequest struct {
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	Role      string  `json:"role"`
	ExpiresAt *string `json:"expires_at"`
}

func (h *AuthHandler) CreateAPIToken(c echo.Context) error {
	actor, _ := GetAuthUser(c)
	var req createAPITokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	userID, err := parseUUID(req.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user_id")
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "expires_at must be RFC3339")
		}
		expiresAt = &t
	}
	if req.Role == "" {
		req.Role = service.RoleViewer
	}
	result, err := h.authSvc.CreateAPIToken(c.Request().Context(), actor, service.CreateAPITokenInput{
		UserID:    userID,
		Name:      req.Name,
		Role:      req.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidRole) || errors.Is(err, service.ErrTokenRoleExceedsOwner) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, service.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"token":     result.PlaintextToken,
		"api_token": safeToken(result.Token),
	})
}

func (h *AuthHandler) DeleteAPIToken(c echo.Context) error {
	actor, _ := GetAuthUser(c)
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid token id")
	}
	if err := h.authSvc.DeleteAPIToken(c.Request().Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrTokenNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "token not found")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ---- Audit log ----

func (h *AuthHandler) ListAuditLog(c echo.Context) error {
	var params db.ListAuditLogsParams
	params.Limit = 100
	if v := c.QueryParam("limit"); v != "" {
		params.Limit = parseIntDefault(v, 100)
	}
	params.Offset = parseIntDefault(c.QueryParam("offset"), 0)
	if v := c.QueryParam("actor_user_id"); v != "" {
		if uid, err := parseUUID(v); err == nil {
			params.ActorUserID = uid
		}
	}
	if v := c.QueryParam("action"); v != "" {
		params.Action = pgtype.Text{String: v, Valid: true}
	}
	if v := c.QueryParam("object_type"); v != "" {
		params.ObjectType = pgtype.Text{String: v, Valid: true}
	}
	if v := c.QueryParam("object_id"); v != "" {
		if uid, err := parseUUID(v); err == nil {
			params.ObjectID = uid
		}
	}
	entries, err := h.auditSvc.List(c.Request().Context(), params)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, entries)
}

// ---- serialization helpers ----

func safeUser(u db.User) map[string]any {
	return map[string]any{
		"id":            uuidStr(u.ID),
		"email":         u.Email,
		"display_name":  u.DisplayName,
		"role":          u.Role,
		"enabled":       u.Enabled,
		"last_login_at": timeOrNilTZ(u.LastLoginAt),
		"created_at":    timeOrNilTZ(u.CreatedAt),
		"updated_at":    timeOrNilTZ(u.UpdatedAt),
	}
}

func safeToken(t db.ApiToken) map[string]any {
	return map[string]any{
		"id":           uuidStr(t.ID),
		"user_id":      uuidStr(t.UserID),
		"name":         t.Name,
		"role":         t.Role,
		"last_used_at": timeOrNilTZ(t.LastUsedAt),
		"expires_at":   timeOrNilTZ(t.ExpiresAt),
		"created_at":   timeOrNilTZ(t.CreatedAt),
	}
}
