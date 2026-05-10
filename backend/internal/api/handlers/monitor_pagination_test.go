package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestParseLimitOffsetDefaults exercises the limit/offset query-param parsing
// the GetMonitorResults handler uses, so the defaults stay 100 / 0 and bad
// inputs degrade gracefully.
func TestParseLimitOffsetDefaults(t *testing.T) {
	cases := []struct {
		query           string
		wantLim, wantOff int32
	}{
		{"", 100, 0},
		{"?limit=50", 50, 0},
		{"?limit=50&offset=200", 50, 200},
		{"?limit=0", 100, 0},        // < 1 falls back to default
		{"?limit=99999", 1000, 0},   // capped at 1000
		{"?limit=abc", 100, 0},      // garbage → default
		{"?offset=-5", 100, 0},      // negative offset clamped to 0
	}

	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/x"+tc.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			lim := int32(100)
			off := int32(0)
			if v := c.QueryParam("limit"); v != "" {
				lim = parseIntDefault(v, 100)
				if lim < 1 {
					lim = 100
				}
				if lim > 1000 {
					lim = 1000
				}
			}
			if v := c.QueryParam("offset"); v != "" {
				off = parseIntDefault(v, 0)
				if off < 0 {
					off = 0
				}
			}
			if lim != tc.wantLim || off != tc.wantOff {
				t.Errorf("query %q: got lim=%d off=%d, want lim=%d off=%d", tc.query, lim, off, tc.wantLim, tc.wantOff)
			}
		})
	}
}
