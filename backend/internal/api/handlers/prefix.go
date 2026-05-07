package handlers

import (
	"net/http"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (h *Handlers) GetPrefixes(c echo.Context) error {
	prefixes, err := h.PrefixService.GetPrefixes(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, prefixes)
}

func (h *Handlers) GetPrefix(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	prefix, err := h.PrefixService.GetPrefix(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "prefix not found")
	}
	return c.JSON(http.StatusOK, prefix)
}

func (h *Handlers) CreatePrefix(c echo.Context) error {
	req := new(db.CreatePrefixParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	prefix, err := h.PrefixService.CreatePrefix(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, prefix)
}

func (h *Handlers) UpdatePrefix(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	req := new(db.UpdatePrefixParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.ID = id

	prefix, err := h.PrefixService.UpdatePrefix(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, prefix)
}

func (h *Handlers) DeletePrefix(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.PrefixService.DeletePrefix(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) GetNextAvailableIP(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	ip, err := h.PrefixService.NextAvailableIP(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"ip_address": ip,
	})
}
