package resources

import "embed"

//go:embed ansible/collect.yaml catalog/*.yaml
var Files embed.FS
