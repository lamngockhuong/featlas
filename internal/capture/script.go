// Package capture runs capture scripts with chromedp and writes screenshots
// with element boxes measured in the same DOM state (capture.json).
package capture

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Script is a capture.yaml: enough steps to sign in and take screenshots.
// The Playwright and pinchtab scripts of the muninn-gen skill read the same file.
type Script struct {
	BaseURL  string `yaml:"baseUrl"`
	Viewport struct {
		W int `yaml:"w"`
		H int `yaml:"h"`
	} `yaml:"viewport"`
	// DeviceScaleFactor is the devicePixelRatio to emulate. Default 1.
	DeviceScaleFactor float64 `yaml:"deviceScaleFactor"`
	// MaxWidth caps the exported image width; larger shots are scaled down
	// and boxes scaled with them. Default 1440.
	MaxWidth int    `yaml:"maxWidth"`
	Steps    []Step `yaml:"steps"`
}

// Step holds exactly one action.
type Step struct {
	Goto    string       `yaml:"goto,omitempty"`
	Fill    *FieldValue  `yaml:"fill,omitempty"`
	Click   string       `yaml:"click,omitempty"`
	Select  *FieldValue  `yaml:"select,omitempty"`
	Upload  *Upload      `yaml:"upload,omitempty"`
	WaitFor *Wait        `yaml:"waitFor,omitempty"`
	Capture *CaptureStep `yaml:"capture,omitempty"`
}

type FieldValue struct {
	Selector string `yaml:"selector"`
	Value    string `yaml:"value"`
}

type Upload struct {
	Selector string `yaml:"selector"`
	File     string `yaml:"file"` // relative to the script directory
}

// Wait waits for one condition: a visible selector, a URL substring, or a delay.
type Wait struct {
	Selector string `yaml:"selector,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Ms       int    `yaml:"ms,omitempty"`
}

// CaptureStep measures elements and takes a screenshot of the same DOM state.
type CaptureStep struct {
	// Out is the output path without extension, relative to the script
	// directory. Writes <out>.webp and <out>.capture.json.
	Out      string            `yaml:"out"`
	Elements map[string]string `yaml:"elements"` // key -> CSS selector
	// Crop is null, a CSS selector, or [x, y, w, h] in CSS pixels.
	Crop any `yaml:"crop"`
}

// Element is one key/selector pair in a stable order.
type Element struct{ Key, Selector string }

// SortedElements returns the elements ordered by key, the order every
// backend writes them in capture.json.
func (c *CaptureStep) SortedElements() []Element {
	out := make([]Element, 0, len(c.Elements))
	for k, s := range c.Elements {
		out = append(out, Element{k, s})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// CropRect returns the crop as a selector or a CSS-pixel rectangle.
func (c *CaptureStep) CropRect() (selector string, rect []float64, err error) {
	switch v := c.Crop.(type) {
	case nil:
		return "", nil, nil
	case string:
		return v, nil, nil
	case []any:
		if len(v) != 4 {
			return "", nil, errors.New("crop must be [x, y, w, h]")
		}
		for _, n := range v {
			switch f := n.(type) {
			case int:
				rect = append(rect, float64(f))
			case float64:
				rect = append(rect, f)
			default:
				return "", nil, errors.New("crop must be [x, y, w, h] numbers")
			}
		}
		if rect[2] <= 0 || rect[3] <= 0 {
			return "", nil, errors.New("crop width and height must be positive")
		}
		return "", rect, nil
	}
	return "", nil, errors.New("crop must be null, a selector, or [x, y, w, h]")
}

var envRef = regexp.MustCompile(`\$\{env:([A-Za-z_][A-Za-z0-9_]*)\}`)

// LoadScript parses a capture script, substitutes ${env:NAME} references and
// checks every step. Missing environment variables fail here, before any
// browser starts, so secrets never need to be written in the file.
func LoadScript(path string) (*Script, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseScript(raw, os.LookupEnv)
}

// ParseScript is LoadScript on bytes with an injectable environment lookup.
func ParseScript(raw []byte, lookup func(string) (string, bool)) (*Script, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	missing := map[string]bool{}
	expandEnv(&root, lookup, missing)
	if len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for n := range missing {
			names = append(names, n)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("environment variables not set: %s", strings.Join(names, ", "))
	}
	s := &Script{}
	if err := root.Decode(s); err != nil {
		return nil, err
	}
	if s.DeviceScaleFactor == 0 {
		s.DeviceScaleFactor = 1
	}
	if s.MaxWidth == 0 {
		s.MaxWidth = 1440
	}
	return s, s.check()
}

func expandEnv(n *yaml.Node, lookup func(string) (string, bool), missing map[string]bool) {
	if n.Kind == yaml.ScalarNode {
		before := n.Value
		n.Value = envRef.ReplaceAllStringFunc(n.Value, func(m string) string {
			name := envRef.FindStringSubmatch(m)[1]
			v, ok := lookup(name)
			if !ok {
				missing[name] = true
			}
			return v
		})
		if n.Value != before && n.Style == 0 {
			n.Tag = "" // re-resolve, so ${env:W} can fill an integer field
		}
		return
	}
	for _, c := range n.Content {
		expandEnv(c, lookup, missing)
	}
}

func (s *Script) check() error {
	if s.Viewport.W <= 0 || s.Viewport.H <= 0 {
		return errors.New("viewport { w, h } is required")
	}
	if s.DeviceScaleFactor < 0 || s.MaxWidth < 0 {
		return errors.New("deviceScaleFactor and maxWidth must be positive")
	}
	if len(s.Steps) == 0 {
		return errors.New("steps is empty")
	}
	for i, st := range s.Steps {
		n := 0
		for _, set := range []bool{st.Goto != "", st.Fill != nil, st.Click != "", st.Select != nil, st.Upload != nil, st.WaitFor != nil, st.Capture != nil} {
			if set {
				n++
			}
		}
		if n != 1 {
			return fmt.Errorf("step %d: needs exactly one of goto, fill, click, select, upload, waitFor, capture", i+1)
		}
		if w := st.WaitFor; w != nil && btoi(w.Selector != "")+btoi(w.URL != "")+btoi(w.Ms > 0) != 1 {
			return fmt.Errorf("step %d: waitFor needs exactly one of selector, url, ms", i+1)
		}
		if st.Upload != nil && !inside(st.Upload.File) {
			return fmt.Errorf("step %d: upload.file must be a path inside the script directory", i+1)
		}
		if c := st.Capture; c != nil {
			if c.Out == "" {
				return fmt.Errorf("step %d: capture.out is required", i+1)
			}
			if !inside(c.Out) {
				return fmt.Errorf("step %d: capture.out must be a path inside the script directory", i+1)
			}
			if _, _, err := c.CropRect(); err != nil {
				return fmt.Errorf("step %d: %w", i+1, err)
			}
		}
	}
	return nil
}

// inside reports whether a script-relative path stays in the script
// directory, so a capture script cannot read or overwrite files elsewhere.
func inside(p string) bool {
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		return false
	}
	for _, seg := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return false
		}
	}
	return true
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ResolveURL joins a goto target with baseUrl unless it is absolute.
func (s *Script) ResolveURL(target string) string {
	if u, err := url.Parse(target); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return target
	}
	return strings.TrimRight(s.BaseURL, "/") + "/" + strings.TrimLeft(target, "/")
}
