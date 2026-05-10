package handlers

import (
	"net/http"

	"blackgrid/internal/version"

	"github.com/labstack/echo/v4"
)

func (h *Handlers) Health(c echo.Context) error {
	v := version.Get()
	return c.JSON(http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    v.Version,
		"commit":     v.Commit,
		"build_date": v.BuildDate,
	})
}
