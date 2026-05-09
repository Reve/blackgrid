package service

import (
	"context"
	"testing"
)

// testCtx returns a context tied to the test's lifetime.
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}
