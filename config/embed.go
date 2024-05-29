package config

import (
	"embed"
)

//go:embed revcat.toml
var ConfigFS embed.FS
