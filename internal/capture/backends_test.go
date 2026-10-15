package capture_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lamngockhuong/muninn/internal/capture"
	"github.com/lamngockhuong/muninn/internal/testutil"
)

// TestBackendsAgree runs examples/demo/capture.yaml through every available
// backend and requires the same elements with boxes within 2px. It needs a
// browser, so it only runs with MUNINN_CAPTURE_E2E=1. Playwright runs when
// node and the skill's node_modules are present; pinchtab when PINCHTAB_URL
// is set (with PINCHTAB_TOKEN and evaluation enabled on the server).
func TestBackendsAgree(t *testing.T) {
	if os.Getenv("MUNINN_CAPTURE_E2E") != "1" {
		t.Skip("set MUNINN_CAPTURE_E2E=1 to run the browser capture comparison")
	}
	root := testutil.RepoRoot()
	app := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(root, "examples", "demo", "app"))))
	defer app.Close()
	t.Setenv("ACME_PASSWORD", "demo")
	src, err := os.ReadFile(filepath.Join(root, "examples", "demo", "capture.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Replace(string(src), "baseUrl: http://localhost:5173", "baseUrl: "+app.URL, 1)
	work := t.TempDir()
	prepare := func(name string) string {
		dir := filepath.Join(work, name)
		testutil.Write(t, filepath.Join(dir, "capture.yaml"), script)
		return dir
	}

	bin := filepath.Join(work, "muninn")
	gobuild := exec.Command("go", "build", "-o", bin, "./cmd/muninn")
	gobuild.Dir = root
	if out, err := gobuild.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	results := map[string]string{}

	dir := prepare("chromedp")
	s, err := capture.LoadScript(filepath.Join(dir, "capture.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := capture.Run(ctx, s, dir, capture.Options{}); err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	results["chromedp"] = dir

	scripts := filepath.Join(root, "plugins", "muninn", "skills", "muninn-gen", "scripts")
	node, nodeErr := exec.LookPath("node")
	if _, err := os.Stat(filepath.Join(scripts, "node_modules", "playwright")); nodeErr == nil && err == nil {
		dir := prepare("playwright")
		runNode(t, node, filepath.Join(scripts, "capture-playwright.mjs"), dir, bin)
		results["playwright"] = dir
	} else {
		t.Log("skipping Playwright: run npm install in plugins/muninn/skills/muninn-gen/scripts")
	}
	if os.Getenv("PINCHTAB_URL") != "" && nodeErr == nil {
		dir := prepare("pinchtab")
		runNode(t, node, filepath.Join(scripts, "capture-pinchtab.mjs"), dir, bin)
		results["pinchtab"] = dir
	} else {
		t.Log("skipping pinchtab: set PINCHTAB_URL")
	}

	ref := loadResults(t, results["chromedp"])
	if len(ref) != 9 {
		t.Fatalf("chromedp wrote %d capture files, want 9", len(ref))
	}
	for name, dir := range results {
		got := loadResults(t, dir)
		for file, want := range ref {
			res, ok := got[file]
			if !ok {
				t.Errorf("%s: missing %s", name, file)
				continue
			}
			if problems, err := capture.ValidateFile(filepath.Join(dir, file)); err != nil || len(problems) > 0 {
				t.Errorf("%s: %s invalid: %v %v", name, file, err, problems)
			}
			if res.Backend != name || len(res.Elements) != len(want.Elements) {
				t.Errorf("%s: %s backend %q, %d elements, want %d", name, file, res.Backend, len(res.Elements), len(want.Elements))
				continue
			}
			for i, e := range res.Elements {
				w := want.Elements[i]
				for j := range 4 {
					if d := e.Box[j] - w.Box[j]; d > 2 || d < -2 {
						t.Errorf("%s: %s %s box %v differs from chromedp %v by more than 2px", name, file, e.Key, e.Box, w.Box)
						break
					}
				}
			}
		}
	}
}

func runNode(t *testing.T, node, script, dir, bin string) {
	t.Helper()
	cmd := exec.Command(node, script, filepath.Join(dir, "capture.yaml"))
	cmd.Env = append(os.Environ(), "MUNINN_BIN="+bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", filepath.Base(script), err, out)
	}
}

func loadResults(t *testing.T, dir string) map[string]capture.Result {
	t.Helper()
	out := map[string]capture.Result{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(p, ".capture.json") {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var r capture.Result
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[rel] = r
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
