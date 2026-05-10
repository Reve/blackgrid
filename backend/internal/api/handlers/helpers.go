package handlers

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"blackgrid/internal/service"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

// AuditFromContext builds the actor/request fields of an AuditParams from an
// Echo context: actor user ID and type from the auth middleware, the request
// ID stamped by the request-ID middleware, and the parsed remote IP. Callers
// fill in Action / EntityType / EntityID / Before / After.
func AuditFromContext(c echo.Context) service.AuditParams {
	p := service.AuditParams{
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		ActorType: "system",
	}
	if u, ok := GetAuthUser(c); ok {
		p.ActorUserID = u.ID
		p.ActorType = "user"
	}
	if tokID, ok := c.Get(ctxKeyTokenID).(pgtype.UUID); ok && tokID.Valid {
		p.ActorTokenID = tokID
		p.ActorType = "api_token"
	}
	if ip := remoteIP(c); ip != nil {
		p.IPAddress = ip
	}
	return p
}

// LogAudit merges the actor/request fields from the Echo context into p and
// dispatches the audit entry. Use this from any handler that touches a
// user-modifiable resource so we get a consistent actor trail.
func LogAudit(audit *service.AuditService, c echo.Context, p service.AuditParams) {
	if audit == nil {
		return
	}
	base := AuditFromContext(c)
	if !p.ActorUserID.Valid {
		p.ActorUserID = base.ActorUserID
	}
	if !p.ActorTokenID.Valid {
		p.ActorTokenID = base.ActorTokenID
	}
	if p.ActorType == "" {
		p.ActorType = base.ActorType
	}
	if p.RequestID == "" {
		p.RequestID = base.RequestID
	}
	if p.IPAddress == nil {
		p.IPAddress = base.IPAddress
	}
	audit.Log(c.Request().Context(), p)
}

func remoteIP(c echo.Context) *netip.Addr {
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		host = c.Request().RemoteAddr
	}
	if host == "" {
		return nil
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	return &addr
}

// parseUUID parses a UUID string into pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if s == "" {
		return id, fmt.Errorf("empty uuid")
	}
	if err := id.Scan(s); err != nil {
		return id, err
	}
	return id, nil
}

// uuidStr converts pgtype.UUID to string representation.
func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// timeOrNilTZ returns nil for invalid timestamps or an RFC3339 string.
func timeOrNilTZ(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// parseIntDefault parses s as int32, returning def on failure.
func parseIntDefault(s string, def int32) int32 {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return int32(n)
}
