//go:build dev

package webui

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// Handler proxies to the Vite dev server (MUNINN_VITE_URL, default
// http://localhost:5174) so the UI hot-reloads while the API is served by Go.
func Handler() http.Handler {
	target := os.Getenv("MUNINN_VITE_URL")
	if target == "" {
		target = "http://localhost:5174"
	}
	u, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

// Files is empty in dev builds: the app lives in the Vite dev server, so the
// static export needs a regular build.
func Files() fs.FS { return emptyFS{} }

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }
