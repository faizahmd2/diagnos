package resources

import "embed"

//go:embed catalog/*.yaml
var Files embed.FS
