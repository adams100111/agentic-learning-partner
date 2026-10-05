package schemas

import "embed"

// Files contains ALP's versioned JSON Schema contracts.
//
//go:embed *.schema.json
var Files embed.FS
