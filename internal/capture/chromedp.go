package capture

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

//go:embed measure.js
var measureJS string

// Options configure a chromedp run.
type Options struct {
	Chrome  string // browser executable; found automatically when empty
	Profile string // user data dir kept between runs; temporary when empty
	Headful bool
	Timeout time.Duration // per step
	Log     io.Writer
}

// FindChrome locates a Chrome or Chromium executable without downloading
// anything: MUNINN_CHROME, well-known names on PATH, standard install
// locations, then a Chromium already installed by Playwright.
func FindChrome() (string, error) {
	if p := os.Getenv("MUNINN_CHROME"); p != "" {
		return p, nil
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			candidates = append(candidates, filepath.Join(os.Getenv(env), `Google\Chrome\Application\chrome.exe`))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, pattern := range []string{
			".cache/ms-playwright/chromium-*/chrome-linux*/chrome",
			"Library/Caches/ms-playwright/chromium-*/chrome-mac*/Chromium.app/Contents/MacOS/Chromium",
			`AppData\Local\ms-playwright\chromium-*\chrome-win*\chrome.exe`,
		} {
			matches, _ := filepath.Glob(filepath.Join(home, pattern))
			for i := len(matches) - 1; i >= 0; i-- { // newest revision first
				candidates = append(candidates, matches[i])
			}
		}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", errors.New("no Chrome or Chromium found. Install Google Chrome (https://www.google.com/chrome/) " +
		"or Chromium from your package manager, or pass --chrome <path> / set MUNINN_CHROME")
}

type measurement struct {
	DPR      float64      `json:"dpr"`
	URL      string       `json:"url"`
	Elements []RawElement `json:"elements"`
	Crop     *struct {
		Missing    bool `json:"missing"`
		X, Y, W, H float64
	} `json:"crop"`
}

// Run executes the script with chromedp. Output paths are relative to dir.
func Run(ctx context.Context, s *Script, dir string, opt Options) ([]string, error) {
	chrome := opt.Chrome
	if chrome == "" {
		var err error
		if chrome, err = FindChrome(); err != nil {
			return nil, err
		}
	}
	if opt.Timeout == 0 {
		opt.Timeout = 30 * time.Second
	}
	if opt.Log == nil {
		opt.Log = io.Discard
	}
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chrome),
		chromedp.Flag("headless", !opt.Headful),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.WindowSize(s.Viewport.W, s.Viewport.H),
	)
	if opt.Profile != "" {
		abs, err := filepath.Abs(opt.Profile)
		if err != nil {
			return nil, err
		}
		allocOpts = append(allocOpts, chromedp.UserDataDir(abs))
	}
	actx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOpts...)
	defer cancelAlloc()
	bctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	if err := chromedp.Do(bctx, chromedp.EmulateViewport(int64(s.Viewport.W), int64(s.Viewport.H), chromedp.EmulateScale(s.DeviceScaleFactor))); err != nil {
		return nil, fmt.Errorf("start browser %s: %w", chrome, err)
	}
	var written []string
	for i, st := range s.Steps {
		sctx, scancel := context.WithTimeout(bctx, opt.Timeout)
		out, err := runStep(sctx, s, dir, st, opt.Log)
		scancel()
		if err != nil {
			return written, fmt.Errorf("step %d: %w", i+1, err)
		}
		written = append(written, out...)
	}
	return written, nil
}

func runStep(ctx context.Context, s *Script, dir string, st Step, log io.Writer) ([]string, error) {
	switch {
	case st.Goto != "":
		fmt.Fprintln(log, "goto", st.Goto)
		return nil, chromedp.Do(ctx, chromedp.Navigate(s.ResolveURL(st.Goto)))
	case st.Fill != nil:
		return nil, fill(ctx, st.Fill)
	case st.Click != "":
		return nil, chromedp.Do(ctx, chromedp.Click(chromedp.CSS(st.Click), chromedp.NodeVisible))
	case st.Select != nil:
		sel := chromedp.CSS(st.Select.Selector)
		return nil, chromedp.Do(ctx, chromedp.WaitVisible(sel), chromedp.SetValue(sel, st.Select.Value),
			chromedp.Evaluate[chromedp.Void](fmt.Sprintf(`void document.querySelector(%s).dispatchEvent(new Event('change', {bubbles: true}))`, jsString(st.Select.Selector))))
	case st.Upload != nil:
		file, err := filepath.Abs(filepath.Join(dir, filepath.FromSlash(st.Upload.File)))
		if err != nil {
			return nil, err
		}
		return nil, chromedp.Do(ctx, chromedp.SetUploadFiles(chromedp.CSS(st.Upload.Selector), []string{file}))
	case st.WaitFor != nil:
		return nil, waitFor(ctx, st.WaitFor)
	case st.Capture != nil:
		return capture(ctx, s, dir, st.Capture, log)
	}
	return nil, errors.New("empty step")
}

// fill types text like a user. Date and similar pickers take keystrokes in
// a locale-dependent order, so their value is set directly, with the same
// input and change events a user edit fires.
func fill(ctx context.Context, f *FieldValue) error {
	sel := chromedp.CSS(f.Selector)
	if err := chromedp.Do(ctx, chromedp.WaitVisible(sel)); err != nil {
		return err
	}
	typ, err := chromedp.Run(ctx, chromedp.Evaluate[string](fmt.Sprintf(`document.querySelector(%s).type || ""`, jsString(f.Selector))))
	if err != nil {
		return err
	}
	switch typ {
	case "date", "time", "datetime-local", "month", "week", "color", "range":
		return chromedp.Do(ctx, chromedp.Evaluate[chromedp.Void](fmt.Sprintf(
			`void (() => { const el = document.querySelector(%s); el.value = %s; el.dispatchEvent(new Event('input', {bubbles: true})); el.dispatchEvent(new Event('change', {bubbles: true})); })()`,
			jsString(f.Selector), jsString(f.Value))))
	}
	return chromedp.Do(ctx, chromedp.Clear(sel), chromedp.SendKeys(sel, f.Value))
}

func waitFor(ctx context.Context, w *Wait) error {
	switch {
	case w.Selector != "":
		return chromedp.Do(ctx, chromedp.WaitVisible(chromedp.CSS(w.Selector)))
	case w.URL != "":
		for {
			href, err := chromedp.Run(ctx, chromedp.Location())
			if err != nil {
				return err
			}
			if strings.Contains(href, w.URL) {
				return nil
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("waiting for URL containing %q (at %s): %w", w.URL, href, ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
	default:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(w.Ms) * time.Millisecond):
			return nil
		}
	}
}

func capture(ctx context.Context, s *Script, dir string, c *CaptureStep, log io.Writer) ([]string, error) {
	cropSel, cropRect, _ := c.CropRect()
	elems := c.SortedElements()
	pairs := make([][2]string, len(elems))
	for i, e := range elems {
		pairs[i] = [2]string{e.Key, e.Selector}
	}
	spec, _ := json.Marshal(map[string]any{"elements": pairs, "cropSelector": cropSel})
	expr := "(" + strings.TrimSpace(stripComments(measureJS)) + ")(" + string(spec) + ")"
	var m measurement
	var shot []byte
	// Measure, then screenshot, with no other action in between.
	err := chromedp.Do(ctx, chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		var err error
		if m, err = chromedp.Evaluate[measurement](expr, chromedp.EvalAwaitPromise)(ctx, t); err != nil {
			return err
		}
		shot, err = chromedp.CaptureScreenshot()(ctx, t)
		return err
	}))
	if err != nil {
		return nil, err
	}
	in := Raw{
		Backend: "chromedp", URL: m.URL, Viewport: Viewport{W: s.Viewport.W, H: s.Viewport.H},
		DPR: m.DPR, MaxWidth: s.MaxWidth, Out: filepath.Join(dir, filepath.FromSlash(c.Out)), Elements: m.Elements,
	}
	switch {
	case cropRect != nil:
		in.Crop = &Rect{cropRect[0], cropRect[1], cropRect[2], cropRect[3]}
	case m.Crop != nil && m.Crop.Missing:
		return nil, fmt.Errorf("crop selector %q matches nothing", cropSel)
	case m.Crop != nil:
		in.Crop = &Rect{m.Crop.X, m.Crop.Y, m.Crop.W, m.Crop.H}
	}
	res, err := Finalize(in, shot)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(log, "captured %s (%d elements)\n", c.Out+".webp", len(res.Elements))
	return []string{in.Out + ".webp", in.Out + ".capture.json"}, nil
}

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// stripComments drops leading // comment lines of the embedded script.
func stripComments(js string) string {
	var lines []string
	for _, l := range strings.Split(js, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "//") {
			continue
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n")
}
