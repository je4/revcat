package web

import "embed"

//go:embed css/* images/* js/*
var FS embed.FS
