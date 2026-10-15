package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const missingUI = `<!doctype html><meta charset="utf-8"><title>muninn</title>
<p>The web UI is not part of this binary. Run <code>make web</code> and rebuild, or use a release binary.
The API is available under <a href="/api/project">/api/project</a>.</p>`

// SPA serves files from fsys and falls back to index.html for paths without
// a file, so client-side routes survive a reload. Hashed assets are cached
// for a year; index.html is always revalidated.
func SPA(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p != "" && p != "index.html" {
			if st, err := fs.Stat(fsys, p); err == nil && !st.IsDir() {
				if strings.HasPrefix(p, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
			if path.Ext(p) != "" {
				http.NotFound(w, r)
				return
			}
		}
		index, err := fs.ReadFile(fsys, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if err != nil {
			_, _ = w.Write([]byte(missingUI))
			return
		}
		_, _ = w.Write(index)
	})
}
