// Package buildinfo holds release metadata injected at link time, e.g.
//
//	go build -ldflags "-X .../internal/buildinfo.Version=v0.1.0"
package buildinfo

// These are variables (not constants) so -ldflags -X can override them.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
