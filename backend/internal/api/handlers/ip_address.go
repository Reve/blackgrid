package handlers

import (
	"net/http"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (h *Handlers) GetIPAddresses(c echo.Context) error {
	ips, err := h.IPAddressService.GetIPAddresses(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, ips)
}

func (h *Handlers) GetIPAddress(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	ip, err := h.IPAddressService.GetIPAddress(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ip address not found")
	}
	return c.JSON(http.StatusOK, ip)
}

func (h *Handlers) CreateIPAddress(c echo.Context) error {
	req := new(db.CreateIPAddressParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ip, err := h.IPAddressService.CreateIPAddress(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, ip)
}

func (h *Handlers) UpdateIPAddress(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	req := new(db.UpdateIPAddressParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.ID = id

	ip, err := h.IPAddressService.UpdateIPAddress(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, ip)
}

// ReserveIPAddress sets status="reserved".
func (h *Handlers) ReserveIPAddress(c echo.Context) error {
	return h.setIPStatus(c, "reserved")
}

// AssignIPAddress sets status="assigned".
func (h *Handlers) AssignIPAddress(c echo.Context) error {
	return h.setIPStatus(c, "assigned")
}

// ReleaseIPAddress sets status="available".
func (h *Handlers) ReleaseIPAddress(c echo.Context) error {
	return h.setIPStatus(c, "available")
}

func (h *Handlers) setIPStatus(c echo.Context, status string) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	ip, err := h.IPAddressService.SetStatus(c.Request().Context(), id, status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, ip)
}

func (h *Handlers) DeleteIPAddress(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.IPAddressService.DeleteIPAddress(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
