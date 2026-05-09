package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
