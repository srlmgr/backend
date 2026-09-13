package server

import "embed"

// staticFiles contains browser assets served by the HTML server.
//
//go:embed static/*.css
var staticFiles embed.FS
