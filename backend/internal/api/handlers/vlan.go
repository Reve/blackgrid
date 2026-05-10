package handlers

import (
	"net/http"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (h *Handlers) GetVlans(c echo.Context) error {
	vlans, err := h.VlanService.GetVlans(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, vlans)
}

func (h *Handlers) GetVlan(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	vlan, err := h.VlanService.GetVlan(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "vlan not found")
	}
	return c.JSON(http.StatusOK, vlan)
}

func (h *Handlers) CreateVlan(c echo.Context) error {
	req := new(db.CreateVlanParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	vlan, err := h.VlanService.CreateVlan(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusCreated, vlan)
}

func (h *Handlers) UpdateVlan(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	req := new(db.UpdateVlanParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.ID = id

	vlan, err := h.VlanService.UpdateVlan(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, vlan)
}

func (h *Handlers) DeleteVlan(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.VlanService.DeleteVlan(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.NoContent(http.StatusNoContent)
}
