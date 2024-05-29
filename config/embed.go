package config

import (
	"embed"
)

//go:embed revcat.toml.template
var ConfigFS embed.FS
