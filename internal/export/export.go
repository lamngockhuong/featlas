// Package export writes a read-only static copy of the atlas: the web app,
// every API response as a JSON file, and the content images.
package export

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamngockhuong/muninn/internal/server"
)

// Marker identifies a directory written by export, so a later export may
// replace it without risking unrelated files.
const Marker = ".muninn-export"

// staticMeta switches the app to static mode: relative JSON files, hash
// routing, no live reload and no reviewing.
const staticMeta = `<meta name="muninn-static" content="1">`

// Options configure an export.
type Options struct {
	Out     string
	UI      fs.FS
	Version string
}

// Result summarises what was written.
type Result struct {
	Entities int
	Images   int
	Files    int
}

// Write exports snap into opt.Out. An existing non-empty directory is only
// replaced when an earlier export created it.
func Write(snap *server.Snapshot, opt Options) (Result, error) {
	var res Result
	index, err := fs.ReadFile(opt.UI, "index.html")
	if err != nil {
		return res, errors.New("this binary has no embedded web UI; run `make web` and rebuild, or use a release binary")
	}
	if err := checkOut(opt.Out); err != nil {
		return res, err
	}
	// Build next to the target and swap it in at the end, so a failed export
	// leaves the previous site untouched.
	parent := filepath.Dir(filepath.Clean(opt.Out))
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return res, err
	}
	tmp, err := os.MkdirTemp(parent, "."+filepath.Base(opt.Out)+".tmp-")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmp)
	if err := os.Chmod(tmp, 0o755); err != nil {
		return res, err
	}
	w := writer{root: tmp, res: &res}

	// The app, with relative asset paths so the site works from any sub-path.
	err = fs.WalkDir(opt.UI, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || p == "index.html" || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		data, err := fs.ReadFile(opt.UI, p)
		if err != nil {
			return err
		}
		return w.file(p, data)
	})
	if err != nil {
		return res, err
	}
	html := strings.ReplaceAll(string(index), `="/assets/`, `="./assets/`)
	html = strings.Replace(html, "<head>", "<head>\n    "+staticMeta, 1)
	if err := w.file("index.html", []byte(html)); err != nil {
		return res, err
	}

	caps := server.Capabilities{Static: true, ReadOnlyReason: "static export"}
	if err := w.json("api/project.json", snap.Project(caps, opt.Version)); err != nil {
		return res, err
	}
	if err := w.json("api/entities.json", snap.Summaries(server.Filter{})); err != nil {
		return res, err
	}
	for _, e := range snap.Index.All() {
		if err := w.json(fmt.Sprintf("api/entities/%s/%s.json", e.Kind, e.ID), snap.Detail(e)); err != nil {
			return res, err
		}
		res.Entities++
		for _, img := range e.Images {
			src, err := os.ReadFile(snap.Index.Abs(img))
			if err != nil {
				continue // reported by validation; the page shows a broken image
			}
			if err := w.file("content/"+img, src); err != nil {
				return res, err
			}
			res.Images++
		}
	}
	for name, v := range map[string]any{
		"search-index": snap.SearchIndex(),
		"graph":        snap.Graph(),
		"glossary":     snap.Glossary(),
		"validation":   snap.Validation(),
	} {
		if err := w.json("api/"+name+".json", v); err != nil {
			return res, err
		}
	}
	// GitHub Pages would otherwise hide files and folders starting with _.
	if err := w.file(".nojekyll", nil); err != nil {
		return res, err
	}
	if err := w.file(Marker, []byte("written by muninn export; the whole directory is replaced on the next export\n")); err != nil {
		return res, err
	}
	return res, swap(tmp, opt.Out)
}

// checkOut allows a missing or empty directory, or one a previous export
// wrote. Export owns that directory entirely and replaces it.
func checkOut(out string) error {
	entries, err := os.ReadDir(out)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(entries) == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(out, Marker)); err != nil {
		return fmt.Errorf("%s is not empty and was not written by muninn export; choose an empty directory", out)
	}
	return nil
}

func swap(tmp, out string) error {
	if _, err := os.Stat(out); err == nil {
		old := tmp + ".old"
		if err := os.Rename(out, old); err != nil {
			return err
		}
		if err := os.Rename(tmp, out); err != nil {
			_ = os.Rename(old, out)
			return err
		}
		return os.RemoveAll(old)
	}
	return os.Rename(tmp, out)
}

type writer struct {
	root string
	res  *Result
}

func (w writer) file(rel string, data []byte) error {
	p := filepath.Join(w.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	w.res.Files++
	return os.WriteFile(p, data, 0o644)
}

func (w writer) json(rel string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return w.file(rel, buf.Bytes())
}
