package godomain

import "embed"

// Files contains the bundled Go domain-pack assets.
//
//go:embed competencies.yaml DIAGNOSTIC.md SOURCES.md
var Files embed.FS
