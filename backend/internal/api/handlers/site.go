package handlers

import (
	"net/http"

	"blackgrid/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func (h *Handlers) GetSites(c echo.Context) error {
	sites, err := h.SiteService.GetSites(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, sites)
}

func (h *Handlers) GetSite(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	site, err := h.SiteService.GetSite(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "site not found")
	}
	return c.JSON(http.StatusOK, site)
}

func (h *Handlers) CreateSite(c echo.Context) error {
	req := new(db.CreateSiteParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	site, err := h.SiteService.CreateSite(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusCreated, site)
}

func (h *Handlers) UpdateSite(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	req := new(db.UpdateSiteParams)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.ID = id

	site, err := h.SiteService.UpdateSite(c.Request().Context(), *req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, site)
}

func (h *Handlers) DeleteSite(c echo.Context) error {
	var id pgtype.UUID
	if err := id.Scan(c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.SiteService.DeleteSite(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.NoContent(http.StatusNoContent)
}
