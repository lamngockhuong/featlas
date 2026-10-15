package capture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lamngockhuong/muninn/internal/capture"
	"github.com/lamngockhuong/muninn/internal/testutil"
)

func TestTransformScalesAndCrops(t *testing.T) {
	cases := []struct {
		name     string
		dpr      float64
		shotW    int
		shotH    int
		crop     *capture.Rect
		maxWidth int
		in       capture.Rect
		want     [4]int
		wantW    int
		wantH    int
	}{
		{"identity", 1, 1366, 768, nil, 1440, capture.Rect{X: 81, Y: 126, W: 115, H: 34}, [4]int{81, 126, 115, 34}, 1366, 768},
		{"device pixel ratio 2, scaled to 1440", 2, 2732, 1536, nil, 1440, capture.Rect{X: 100, Y: 50, W: 200, H: 40},
			[4]int{105, 53, 211, 42}, 1440, 810},
		{"crop moves the origin", 1, 1366, 768, &capture.Rect{X: 24, Y: 100, W: 600, H: 300}, 1440, capture.Rect{X: 81, Y: 126, W: 115, H: 34},
			[4]int{57, 26, 115, 34}, 600, 300},
		{"box clipped to the crop", 1, 1366, 768, &capture.Rect{X: 0, Y: 0, W: 100, H: 100}, 1440, capture.Rect{X: 80, Y: 90, W: 50, H: 50},
			[4]int{80, 90, 20, 10}, 100, 100},
		{"fractional CSS pixels round edges", 1.5, 2049, 1152, nil, 0, capture.Rect{X: 10.2, Y: 10.2, W: 20.5, H: 20.5},
			[4]int{15, 15, 31, 31}, 2049, 1152},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, _, err := capture.NewTransform(tc.dpr, tc.shotW, tc.shotH, tc.crop, tc.maxWidth)
			if err != nil {
				t.Fatal(err)
			}
			if tr.W != tc.wantW || tr.H != tc.wantH {
				t.Errorf("image = %dx%d, want %dx%d", tr.W, tr.H, tc.wantW, tc.wantH)
			}
			got, ok := tr.Box(tc.in)
			if !ok || got != tc.want {
				t.Errorf("box = %v %v, want %v", got, ok, tc.want)
			}
		})
	}
	tr, _, _ := capture.NewTransform(1, 1366, 768, nil, 1440)
	if _, ok := tr.Box(capture.Rect{X: 1400, Y: 10, W: 40, H: 10}); ok {
		t.Error("an element outside the screenshot must not be visible")
	}
	if _, _, err := capture.NewTransform(1, 1366, 768, &capture.Rect{X: 2000, Y: 0, W: 10, H: 10}, 0); err == nil {
		t.Error("a crop outside the screenshot must fail")
	}
}

func TestScriptEnvAndSteps(t *testing.T) {
	script := []byte("baseUrl: http://x\nviewport: { w: 800, h: 600 }\nsteps:\n  - fill: { selector: '#p', value: '${env:SECRET_A}-${env:SECRET_B}' }\n")
	_, err := capture.ParseScript(script, func(string) (string, bool) { return "", false })
	if err == nil || !strings.Contains(err.Error(), "SECRET_A, SECRET_B") {
		t.Fatalf("missing env error = %v", err)
	}
	s, err := capture.ParseScript(script, func(n string) (string, bool) { return strings.ToLower(n), true })
	if err != nil {
		t.Fatal(err)
	}
	if v := s.Steps[0].Fill.Value; v != "secret_a-secret_b" {
		t.Errorf("value = %q", v)
	}
	for _, bad := range []string{
		"viewport: { w: 800, h: 600 }\nsteps:\n  - { goto: /, click: a }\n",
		"viewport: { w: 800, h: 600 }\nsteps:\n  - waitFor: { ms: 10, url: /x }\n",
		"viewport: { w: 800, h: 600 }\nsteps:\n  - capture: { elements: {} }\n",
		"steps:\n  - goto: /\n",
	} {
		if _, err := capture.ParseScript([]byte(bad), func(string) (string, bool) { return "", true }); err == nil {
			t.Errorf("accepted:\n%s", bad)
		}
	}
}

// The Node backends must measure with the exact function muninn embeds.
func TestMeasureScriptsIdentical(t *testing.T) {
	root := testutil.RepoRoot()
	a, err := os.ReadFile(filepath.Join(root, "internal", "capture", "measure.js"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "skills", "muninn-gen", "scripts", "measure.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Error("skills/muninn-gen/scripts/measure.js differs from internal/capture/measure.js")
	}
}

func TestValidateFile(t *testing.T) {
	dir := t.TempDir()
	png := mustPNG(t, 100, 50)
	in := capture.Raw{Backend: "playwright", URL: "http://x", Viewport: capture.Viewport{W: 100, H: 50}, DPR: 1, MaxWidth: 1440,
		Out: filepath.Join(dir, "shot"), Elements: []capture.RawElement{{Key: "a", Selector: "#a", X: 10, Y: 10, W: 20, H: 10}}}
	if _, err := capture.Finalize(in, png); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "shot.capture.json")
	if problems, err := capture.ValidateFile(file); err != nil || len(problems) > 0 {
		t.Fatalf("valid file: %v %v", err, problems)
	}
	testutil.Edit(t, file, `"box": [
        10,`, `"box": [
        95,`)
	problems, err := capture.ValidateFile(file)
	if err != nil || len(problems) != 1 || !strings.Contains(problems[0], "lies outside") {
		t.Errorf("box outside image: %v %v", err, problems)
	}
	in.Elements[0].Missing = true
	if _, err := capture.Finalize(in, png); err == nil {
		t.Error("a missing element must fail")
	}
}

func TestScriptConfinementAndURLs(t *testing.T) {
	for _, bad := range []string{
		"viewport: { w: 8, h: 6 }\nsteps:\n  - capture: { out: ../../etc/x }\n",
		"viewport: { w: 8, h: 6 }\nsteps:\n  - capture: { out: /tmp/x }\n",
		"viewport: { w: 8, h: 6 }\nsteps:\n  - upload: { selector: '#f', file: ../secret.txt }\n",
	} {
		if _, err := capture.ParseScript([]byte(bad), func(string) (string, bool) { return "", true }); err == nil {
			t.Errorf("accepted:\n%s", bad)
		}
	}
	s, err := capture.ParseScript([]byte("baseUrl: http://app:3000/\nviewport:\n  w: ${env:W}\n  h: 600\nsteps:\n  - goto: /\n"),
		func(string) (string, bool) { return "1366", true })
	if err != nil {
		t.Fatal(err)
	}
	if s.Viewport.W != 1366 {
		t.Errorf("viewport.w = %d", s.Viewport.W)
	}
	for in, want := range map[string]string{
		"/logout?next=https://x": "http://app:3000/logout?next=https://x",
		"https://sso.example/a":  "https://sso.example/a",
		"claims":                 "http://app:3000/claims",
	} {
		if got := s.ResolveURL(in); got != want {
			t.Errorf("ResolveURL(%q) = %q, want %q", in, got, want)
		}
	}
}
