//go:build !dev

// Package webui serves the React single-page app embedded in the binary.
// `make web` writes the Vite output into static/ before `go build`.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:static
var files embed.FS

// Handler serves the embedded app; unknown paths fall back to index.html so
// client-side routes survive a reload.
func Handler() http.Handler { return SPA(Files()) }

// Files is the embedded app, used by the static export.
func Files() fs.FS {
	sub, _ := fs.Sub(files, "static")
	return sub
}
