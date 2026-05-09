package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

const (
	ErrCodeValidation         = "validation_error"
	ErrCodeUnauthorized       = "unauthorized"
	ErrCodeForbidden          = "forbidden"
	ErrCodeNotFound           = "not_found"
	ErrCodeConflict           = "conflict"
	ErrCodeRateLimited        = "rate_limited"
	ErrCodeInternal           = "internal_error"
	ErrCodeServiceUnavailable = "service_unavailable"
)

// Map HTTP status codes to our internal error codes
func statusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeValidation
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeConflict
	case http.StatusTooManyRequests:
		return ErrCodeRateLimited
	case http.StatusServiceUnavailable:
		return ErrCodeServiceUnavailable
	default:
		return ErrCodeInternal
	}
}

// CustomHTTPErrorHandler handles errors and returns consistent JSON responses
func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Internal Server Error"
	var details map[string]any

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if m, ok := he.Message.(string); ok {
			message = m
		} else {
			message = http.StatusText(code)
		}
	}

	// Use our internal error code
	errCode := statusToCode(code)

	// Don't leak internal errors in production-like environments
	// For now, we'll keep the message if it's not a 500, or if it's explicitly set.
	if code == http.StatusInternalServerError && errCode == ErrCodeInternal {
		// Log the actual error here (we'll add structured logging later)
		c.Logger().Error(err)
		message = "An internal server error occurred"
	}

	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:      errCode,
			Message:   message,
			RequestID: requestID,
			Details:   details,
		},
	}

	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, resp)
		}
		if err != nil {
			c.Logger().Error(err)
		}
	}
}

// Error returns a formatted error response directly from a handler
func Error(c echo.Context, code string, message string, details map[string]any) error {
	status := http.StatusInternalServerError

	switch code {
	case ErrCodeValidation:
		status = http.StatusBadRequest
	case ErrCodeUnauthorized:
		status = http.StatusUnauthorized
	case ErrCodeForbidden:
		status = http.StatusForbidden
	case ErrCodeNotFound:
		status = http.StatusNotFound
	case ErrCodeConflict:
		status = http.StatusConflict
	case ErrCodeRateLimited:
		status = http.StatusTooManyRequests
	case ErrCodeInternal:
		status = http.StatusInternalServerError
	case ErrCodeServiceUnavailable:
		status = http.StatusServiceUnavailable
	}

	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	return c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID,
			Details:   details,
		},
	})
}
