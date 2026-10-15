package export_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lamngockhuong/muninn/internal/export"
	"github.com/lamngockhuong/muninn/internal/server"
	"github.com/lamngockhuong/muninn/internal/testutil"
)

var ui = fstest.MapFS{
	"index.html":     {Data: []byte(`<!doctype html><html><head><script type="module" src="/assets/app.js"></script></head></html>`)},
	"assets/app.js":  {Data: []byte("console.log(1)")},
	".keep":          {Data: nil},
	"favicon.svg":    {Data: []byte("<svg/>")},
	"assets/app.css": {Data: []byte("body{}")},
}

func TestExport(t *testing.T) {
	_, dir := testutil.Demo(t)
	snap, err := server.NewSnapshot(dir, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "site")
	res, err := export.Write(snap, export.Options{Out: out, UI: ui, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entities != 12 || res.Images != 9 {
		t.Errorf("result = %+v", res)
	}
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(index), `<meta name="muninn-static" content="1">`) || !strings.Contains(string(index), `src="./assets/app.js"`) {
		t.Errorf("index.html = %s", index)
	}
	for _, f := range []string{"assets/app.js", "api/entities.json", "api/entities/screen/claim-list.json", "api/search-index.json",
		"api/graph.json", "api/glossary.json", "api/validation.json", "content/screens/claim-list/shots/default.webp", ".nojekyll"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(out, ".keep")); err == nil {
		t.Error(".keep must not be exported")
	}
	var project server.ProjectResponse
	raw, _ := os.ReadFile(filepath.Join(out, "api/project.json"))
	if err := json.Unmarshal(raw, &project); err != nil || !project.Capabilities.Static || project.Capabilities.CanReview || project.Capabilities.LocalRepo != "" {
		t.Errorf("project capabilities = %+v (%v)", project.Capabilities, err)
	}

	// A second export replaces the first; a foreign directory is refused.
	if _, err := export.Write(snap, export.Options{Out: out, UI: ui}); err != nil {
		t.Errorf("re-export: %v", err)
	}
	foreign := t.TempDir()
	testutil.Write(t, filepath.Join(foreign, "keep-me.txt"), "x")
	if _, err := export.Write(snap, export.Options{Out: foreign, UI: ui}); err == nil {
		t.Error("export into a non-empty foreign directory must fail")
	}
	if _, err := os.Stat(filepath.Join(foreign, "keep-me.txt")); err != nil {
		t.Error("foreign file was removed")
	}
}

func TestExportNeedsUI(t *testing.T) {
	_, dir := testutil.Demo(t)
	snap, _ := server.NewSnapshot(dir, nil, 1)
	if _, err := export.Write(snap, export.Options{Out: t.TempDir(), UI: fstest.MapFS{}}); err == nil {
		t.Error("export without a web UI must fail")
	}
}
