package server_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lamngockhuong/muninn/internal/server"
	"github.com/lamngockhuong/muninn/internal/testutil"
	"github.com/lamngockhuong/muninn/internal/webui"
)

func start(t *testing.T, opt server.Options) (*httptest.Server, *server.Server, string, string) {
	t.Helper()
	project, dir := testutil.Demo(t)
	opt.Dir = dir
	opt.UI = webui.Handler()
	srv, err := server.New(opt)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv, project, dir
}

func get(t *testing.T, url string, v any) int {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if v != nil {
		if err := json.NewDecoder(res.Body).Decode(v); err != nil {
			t.Fatalf("%s: %v", url, err)
		}
	}
	return res.StatusCode
}

func post(t *testing.T, url, body string, header map[string]string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestEndpoints(t *testing.T) {
	ts, _, _, _ := start(t, server.Options{Local: true, Version: "test"})
	var project server.ProjectResponse
	if code := get(t, ts.URL+"/api/project", &project); code != 200 || project.Total != 12 || !project.Capabilities.CanReview || project.Capabilities.LocalRepo == "" {
		t.Errorf("project: %d %+v", code, project)
	}
	var list []server.Summary
	get(t, ts.URL+"/api/entities?type=screen&role=approver", &list)
	keys := []string{}
	for _, s := range list {
		keys = append(keys, s.Key)
	}
	if strings.Join(keys, ",") != "screen/approval-queue,screen/claim-detail,screen/sign-in" {
		t.Errorf("filtered screens = %v", keys)
	}
	var detail server.Detail
	if code := get(t, ts.URL+"/api/entities/screen/claim-list", &detail); code != 200 {
		t.Fatalf("detail %d", code)
	}
	if m := detail.Media["shots/default.webp"]; m == nil || m.W != 1366 || m.H != 768 || m.URL != "/content/screens/claim-list/shots/default.webp" {
		t.Errorf("media = %+v", detail.Media)
	}
	if detail.Bodies["vi"] == "" || len(detail.RefsTo) == 0 || detail.Area != "claims" {
		t.Errorf("detail = bodies %v refs %v area %q", detail.Bodies, detail.RefsTo, detail.Area)
	}
	var flow server.Detail
	get(t, ts.URL+"/api/entities/flows/submit-claim", &flow)
	if len(flow.Steps) != 4 || flow.Steps[0].Media == nil || !strings.Contains(flow.Steps[0].Media.URL, "claim-list") {
		t.Errorf("flow steps = %+v", flow.Steps)
	}
	if code := get(t, ts.URL+"/api/entities/screen/nope", nil); code != 404 {
		t.Errorf("missing entity = %d", code)
	}
	var search []server.SearchDoc
	get(t, ts.URL+"/api/search-index", &search)
	if len(search) != 15 { // 12 entities + 3 terms
		t.Errorf("search docs = %d", len(search))
	}
	var graph server.Graph
	get(t, ts.URL+"/api/graph", &graph)
	if len(graph.Nodes) != 12 || len(graph.Edges) == 0 {
		t.Errorf("graph = %d nodes %d edges", len(graph.Nodes), len(graph.Edges))
	}
	var glossary []map[string]any
	get(t, ts.URL+"/api/glossary", &glossary)
	if len(glossary) != 3 {
		t.Errorf("glossary = %v", glossary)
	}
	var validation server.ValidationResponse
	if get(t, ts.URL+"/api/validation", &validation); validation.Errors != 0 {
		t.Errorf("validation = %+v", validation)
	}
	res, _ := http.Get(ts.URL + "/content/screens/claim-list/shots/default.webp")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/webp" {
		t.Errorf("image: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	res.Body.Close()
	res, _ = http.Get(ts.URL + "/screens/claim-list")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "<") {
		t.Errorf("SPA fallback: %d", res.StatusCode)
	}
}

func TestReviewWritesFile(t *testing.T) {
	ts, _, _, dir := start(t, server.Options{})
	code, out := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"Alice"}`, nil)
	if code != 200 || out["status"].(map[string]any)["state"] != "approved" {
		t.Fatalf("review: %d %v", code, out)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "screens/claim-list/screen.md"))
	if !strings.Contains(string(data), "  by: Alice\n") || !strings.Contains(string(data), "contentHash: sha256:") {
		t.Errorf("file not updated:\n%s", data)
	}
	if code, _ := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":""}`, nil); code != 400 {
		t.Errorf("empty name = %d", code)
	}
	if code, _ := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"`+strings.Repeat("x", 101)+`"}`, nil); code != 400 {
		t.Errorf("long name = %d", code)
	}
	if code, _ := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A"}`, map[string]string{"Origin": "https://evil.example"}); code != 403 {
		t.Errorf("cross-origin = %d", code)
	}
}

func TestReviewRefusedWithValidationErrors(t *testing.T) {
	ts, _, _, dir := start(t, server.Options{})
	testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "to: claim-list", "to: nowhere")
	code, out := post(t, ts.URL+"/api/entities/screen/sign-in/review", `{"status":"approved","by":"Alice"}`, nil)
	if code != 422 || out["issues"] == nil {
		t.Errorf("review with errors: %d %v", code, out)
	}
}

func TestWriteProtection(t *testing.T) {
	ro, _, _, _ := start(t, server.Options{ReadOnly: true})
	if code, _ := post(t, ro.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A"}`, nil); code != 403 {
		t.Errorf("read-only = %d", code)
	}
	tok, _, _, _ := start(t, server.Options{Token: "s3cret"})
	if code, _ := post(t, tok.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A"}`, map[string]string{"Authorization": "Bearer wrong"}); code != 401 {
		t.Errorf("wrong token = %d", code)
	}
	if code, _ := post(t, tok.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A"}`, nil); code != 401 {
		t.Errorf("no token = %d", code)
	}
	if code, _ := post(t, tok.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A"}`, map[string]string{"Authorization": "Bearer s3cret"}); code != 200 {
		t.Errorf("right token = %d", code)
	}
}

func TestContentTraversal(t *testing.T) {
	ts, _, project, _ := start(t, server.Options{})
	if err := os.Symlink(filepath.Join(project, "app", "app.js"), filepath.Join(project, "muninn", "screens", "link.js")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/content/../muninn.yaml", "/content/%2e%2e/muninn.yaml", "/content/..%2fapp%2fapp.js", "/content/screens/link.js", "/content/screens"} {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 404 {
			t.Errorf("%s = %d, want 404", p, res.StatusCode)
		}
	}
}

func TestLiveReload(t *testing.T) {
	ts, srv, project, _ := start(t, server.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Watch(ctx) }()
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	time.Sleep(100 * time.Millisecond) // let the watcher register directories
	changed := time.Now()
	testutil.Edit(t, filepath.Join(project, "app/claims.html"), "My claims", "My Claims")
	events := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if sc.Text() == "event: changed" {
				events <- sc.Text()
				return
			}
		}
	}()
	select {
	case <-events:
		if d := time.Since(changed); d > time.Second {
			t.Errorf("event after %v, want under 1s", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no changed event")
	}
}

func TestHostAllowList(t *testing.T) {
	ts, _, _, _ := start(t, server.Options{Local: true, Host: "127.0.0.1"})
	for host, want := range map[string]int{"localhost:7077": 200, "127.0.0.1:7077": 200, "rebind.attacker.example:7077": 421} {
		req, _ := http.NewRequest("GET", ts.URL+"/api/project", nil)
		req.Host = host
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("Host %s = %d, want %d", host, res.StatusCode, want)
		}
	}
}

func TestContentServesImagesOnly(t *testing.T) {
	ts, _, _, dir := start(t, server.Options{})
	testutil.Write(t, filepath.Join(dir, "screens/x.html"), "<script>alert(1)</script>")
	testutil.Write(t, filepath.Join(dir, "screens/x.svg"), "<svg onload=alert(1)/>")
	for _, p := range []string{"/content/screens/x.html", "/content/screens/x.svg", "/content/muninn.yaml"} {
		res, _ := http.Get(ts.URL + p)
		res.Body.Close()
		if res.StatusCode != 404 {
			t.Errorf("%s = %d, want 404", p, res.StatusCode)
		}
	}
	res, _ := http.Get(ts.URL + "/content/screens/claim-list/shots/default.webp")
	res.Body.Close()
	if res.Header.Get("Content-Security-Policy") == "" {
		t.Error("images must be served with a sandboxing CSP")
	}
}

func TestReviewRefusesStaleRevision(t *testing.T) {
	ts, _, _, dir := start(t, server.Options{})
	var detail server.Detail
	get(t, ts.URL+"/api/entities/screen/claim-list", &detail)
	if detail.Revision == "" {
		t.Fatal("no revision in the entity response")
	}
	testutil.Edit(t, filepath.Join(dir, "screens/claim-list/screen.md"), "## Purpose", "## Why")
	code, _ := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A","revision":"`+detail.Revision+`"}`, nil)
	if code != 409 {
		t.Errorf("stale revision = %d, want 409", code)
	}
	get(t, ts.URL+"/api/entities/screen/claim-list", &detail)
	if code, _ := post(t, ts.URL+"/api/entities/screen/claim-list/review", `{"status":"approved","by":"A","revision":"`+detail.Revision+`"}`, nil); code != 200 {
		t.Errorf("current revision = %d, want 200", code)
	}
}
