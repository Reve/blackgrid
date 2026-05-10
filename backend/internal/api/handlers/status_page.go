package handlers

import (
	"errors"
	"net/http"

	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type StatusPageHandler struct {
	svc *service.StatusPageService
}

func NewStatusPageHandler(svc *service.StatusPageService) *StatusPageHandler {
	return &StatusPageHandler{svc: svc}
}

type statusPageRequest struct {
	Name          *string `json:"name"`
	Slug          *string `json:"slug"`
	Description   *string `json:"description"`
	Public        *bool   `json:"public"`
	ShowUptime    *bool   `json:"show_uptime"`
	ShowIncidents *bool   `json:"show_incidents"`
}

func (r statusPageRequest) toInput() service.StatusPageInput {
	in := service.StatusPageInput{
		Description:   r.Description,
		Public:        r.Public,
		ShowUptime:    r.ShowUptime,
		ShowIncidents: r.ShowIncidents,
	}
	if r.Name != nil {
		in.Name = *r.Name
	}
	if r.Slug != nil {
		in.Slug = *r.Slug
	}
	return in
}

func (h *StatusPageHandler) ListStatusPages(c echo.Context) error {
	pages, err := h.svc.ListStatusPages(c.Request().Context())
	if err != nil {
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
	return c.JSON(http.StatusOK, pages)
}

func (h *StatusPageHandler) CreateStatusPage(c echo.Context) error {
	var req statusPageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid body", nil)
	}
	page, err := h.svc.CreateStatusPage(c.Request().Context(), req.toInput())
	if err != nil {
		return statusPageError(c, err)
	}
	return c.JSON(http.StatusCreated, page)
}

func (h *StatusPageHandler) GetStatusPage(c echo.Context) error {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	page, err := h.svc.GetAdminStatusPage(c.Request().Context(), id)
	if err != nil {
		return statusPageError(c, err)
	}
	return c.JSON(http.StatusOK, page)
}

func (h *StatusPageHandler) UpdateStatusPage(c echo.Context) error {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	var req statusPageRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid body", nil)
	}
	page, err := h.svc.UpdateStatusPage(c.Request().Context(), id, req.toInput())
	if err != nil {
		return statusPageError(c, err)
	}
	return c.JSON(http.StatusOK, page)
}

func (h *StatusPageHandler) DeleteStatusPage(c echo.Context) error {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	if err := h.svc.DeleteStatusPage(c.Request().Context(), id); err != nil {
		return statusPageError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

type attachMonitorRequest struct {
	MonitorID    string  `json:"monitor_id"`
	DisplayName  *string `json:"display_name"`
	DisplayOrder *int32  `json:"display_order"`
}

func (h *StatusPageHandler) AttachMonitor(c echo.Context) error {
	pageID, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	var req attachMonitorRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid body", nil)
	}
	var monID pgtype.UUID
	if err := monID.Scan(req.MonitorID); err != nil {
		return Error(c, ErrCodeValidation, "invalid monitor_id", nil)
	}
	link, err := h.svc.AttachMonitor(c.Request().Context(), pageID, service.AttachMonitorInput{
		MonitorID:    monID,
		DisplayName:  req.DisplayName,
		DisplayOrder: req.DisplayOrder,
	})
	if err != nil {
		return statusPageError(c, err)
	}
	return c.JSON(http.StatusCreated, link)
}

type updateAttachedMonitorRequest struct {
	DisplayName  *string `json:"display_name"`
	DisplayOrder *int32  `json:"display_order"`
}

func (h *StatusPageHandler) UpdateAttachedMonitor(c echo.Context) error {
	pageID, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	monID, ok := parseUUIDParam(c, "monitor_id")
	if !ok {
		return nil
	}
	var req updateAttachedMonitorRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid body", nil)
	}
	link, err := h.svc.UpdateAttachedMonitor(c.Request().Context(), pageID, monID, service.UpdateAttachedMonitorInput{
		DisplayName:  req.DisplayName,
		DisplayOrder: req.DisplayOrder,
	})
	if err != nil {
		return statusPageError(c, err)
	}
	return c.JSON(http.StatusOK, link)
}

func (h *StatusPageHandler) RemoveAttachedMonitor(c echo.Context) error {
	pageID, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	monID, ok := parseUUIDParam(c, "monitor_id")
	if !ok {
		return nil
	}
	if err := h.svc.RemoveMonitor(c.Request().Context(), pageID, monID); err != nil {
		return statusPageError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

type reorderRequest struct {
	MonitorIDs []string `json:"monitor_ids"`
}

func (h *StatusPageHandler) ReorderMonitors(c echo.Context) error {
	pageID, ok := parseUUIDParam(c, "id")
	if !ok {
		return nil
	}
	var req reorderRequest
	if err := c.Bind(&req); err != nil {
		return Error(c, ErrCodeValidation, "invalid body", nil)
	}
	ids := make([]pgtype.UUID, 0, len(req.MonitorIDs))
	for _, s := range req.MonitorIDs {
		var u pgtype.UUID
		if err := u.Scan(s); err != nil {
			return Error(c, ErrCodeValidation, "invalid monitor id: " + s, nil)
		}
		ids = append(ids, u)
	}
	if err := h.svc.ReorderMonitors(c.Request().Context(), pageID, ids); err != nil {
		return statusPageError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// PublicStatusPage handles GET /status/:slug.
func (h *StatusPageHandler) PublicStatusPage(c echo.Context) error {
	slug := c.Param("slug")
	page, err := h.svc.GetPublicStatusPage(c.Request().Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrStatusPageNotFound) {
			return Error(c, ErrCodeNotFound, "status page not found", nil)
		}
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
	return c.JSON(http.StatusOK, page)
}

// ----- helpers -----

func parseUUIDParam(c echo.Context, name string) (pgtype.UUID, bool) {
	var id pgtype.UUID
	if err := id.Scan(c.Param(name)); err != nil {
		_ = Error(c, ErrCodeValidation, "invalid " + name, nil)
		return id, false
	}
	return id, true
}

func statusPageError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrStatusPageNotFound):
		return Error(c, ErrCodeNotFound, "status page not found", nil)
	case errors.Is(err, service.ErrStatusPageDuplicateSlug):
		return Error(c, ErrCodeConflict, err.Error(), nil)
	case errors.Is(err, service.ErrStatusPageInvalidSlug),
		errors.Is(err, service.ErrStatusPageNameRequired),
		errors.Is(err, service.ErrReorderMonitorMismatched):
		return Error(c, ErrCodeValidation, err.Error(), nil)
	case errors.Is(err, service.ErrMonitorAlreadyAttached):
		return Error(c, ErrCodeConflict, err.Error(), nil)
	case errors.Is(err, service.ErrMonitorNotAttached):
		return Error(c, ErrCodeNotFound, err.Error(), nil)
	default:
		return Error(c, ErrCodeInternal, "internal error", nil)
	}
}
