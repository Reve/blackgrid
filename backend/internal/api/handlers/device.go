package handlers

import (
	"net/http"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (h *Handlers) GetDevices(c echo.Context) error {
	devices, err := h.DeviceService.GetDevices(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, devices)
}

func (h *Handlers) GetDevice(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	device, err := h.DeviceService.GetDevice(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}
	return c.JSON(http.StatusOK, device)
}

func (h *Handlers) CreateDevice(c echo.Context) error {
	req := new(db.CreateDeviceParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	device, err := h.DeviceService.CreateDevice(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, device)
}

func (h *Handlers) UpdateDevice(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	req := new(db.UpdateDeviceParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.ID = id

	device, err := h.DeviceService.UpdateDevice(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, device)
}

func (h *Handlers) DeleteDevice(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.DeviceService.DeleteDevice(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
