// Package version exposes build-time identity for the running backend.
//
// The exported strings are overridden via -ldflags at build time:
//
//	go build -ldflags "-X blackgrid/internal/version.Version=v0.2.0 \
//	    -X blackgrid/internal/version.Commit=abc1234 \
//	    -X blackgrid/internal/version.BuildDate=2026-05-10T12:00:00Z" \
//	    ./cmd/server
//
// The Docker build accepts VERSION, COMMIT_SHA and BUILD_DATE build args
// and wires them through to these symbols. When unset (e.g. local
// developer builds), the defaults below identify the binary as a dev
// build so operators can tell it apart from a tagged release.
package version

// Version is the human-readable release identifier (e.g. "v0.2.0").
var Version = "dev"

// Commit is the git commit short SHA the binary was built from.
var Commit = "unknown"

// BuildDate is an RFC3339 timestamp of when the binary was built.
var BuildDate = "unknown"

// Info is the structured shape returned by APIs and used by the UI.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// Get returns a snapshot of the build identity.
func Get() Info {
	return Info{Version: Version, Commit: Commit, BuildDate: BuildDate}
}
