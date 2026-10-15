package capture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/draw"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/schema"
)

// Result is the capture.json contract shared by all backends.
type Result struct {
	Backend    string     `json:"backend"`
	URL        string     `json:"url"`
	Viewport   Viewport   `json:"viewport"`
	CapturedAt string     `json:"capturedAt"`
	Shot       string     `json:"shot"`
	Elements   []Measured `json:"elements"`
}

type Viewport struct {
	W int `json:"w"`
	H int `json:"h"`
}

type Measured struct {
	Key      string `json:"key"`
	Selector string `json:"selector"`
	Text     string `json:"text,omitempty"`
	Box      [4]int `json:"box"`
}

// Rect is a rectangle in CSS pixels as returned by getBoundingClientRect.
type Rect struct{ X, Y, W, H float64 }

// Transform maps CSS-pixel rectangles to pixels of the exported image:
// multiply by the device pixel ratio, subtract the crop origin, then scale
// with the resize. Every backend applies the same arithmetic.
type Transform struct {
	DPR   float64
	CropX int // device pixels
	CropY int
	Scale float64 // exported width / cropped width, at most 1
	W, H  int     // exported image size
}

// NewTransform computes the crop (device pixels, clamped to the screenshot)
// and the downscale needed to fit maxWidth.
func NewTransform(dpr float64, shotW, shotH int, crop *Rect, maxWidth int) (Transform, image.Rectangle, error) {
	area := image.Rect(0, 0, shotW, shotH)
	if crop != nil {
		area = image.Rect(
			int(math.Round(crop.X*dpr)), int(math.Round(crop.Y*dpr)),
			int(math.Round((crop.X+crop.W)*dpr)), int(math.Round((crop.Y+crop.H)*dpr)),
		).Intersect(area)
		if area.Empty() {
			return Transform{}, area, fmt.Errorf("crop lies outside the screenshot")
		}
	}
	t := Transform{DPR: dpr, CropX: area.Min.X, CropY: area.Min.Y, Scale: 1, W: area.Dx(), H: area.Dy()}
	if maxWidth > 0 && t.W > maxWidth {
		t.Scale = float64(maxWidth) / float64(t.W)
		t.W = maxWidth
		t.H = int(math.Round(float64(area.Dy()) * t.Scale))
	}
	return t, area, nil
}

// Box converts one CSS rectangle; ok is false when nothing of it is visible
// in the exported image. Edges are rounded independently so adjacent boxes
// stay adjacent after scaling.
func (t Transform) Box(r Rect) (box [4]int, ok bool) {
	conv := func(v float64, origin int) int {
		return int(math.Round((v*t.DPR - float64(origin)) * t.Scale))
	}
	x0, y0 := conv(r.X, t.CropX), conv(r.Y, t.CropY)
	x1, y1 := conv(r.X+r.W, t.CropX), conv(r.Y+r.H, t.CropY)
	x0, y0 = clamp(x0, 0, t.W), clamp(y0, 0, t.H)
	x1, y1 = clamp(x1, 0, t.W), clamp(y1, 0, t.H)
	if x1 <= x0 || y1 <= y0 {
		return box, false
	}
	return [4]int{x0, y0, x1 - x0, y1 - y0}, true
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }

// Export crops and scales a PNG screenshot and encodes it as lossless WebP.
func Export(pngData []byte, area image.Rectangle, t Transform) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}
	img := src
	if area != src.Bounds() || t.Scale != 1 {
		dst := image.NewRGBA(image.Rect(0, 0, t.W, t.H))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, area, draw.Src, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteResult writes <out>.capture.json.
func WriteResult(out string, r *Result) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out+".capture.json", append(data, '\n'), 0o644)
}

// ValidateFile checks a capture.json against its schema and its image: the
// image exists and every box lies inside it. Returns the problems found.
func ValidateFile(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return []string{"invalid JSON: " + err.Error()}, nil
	}
	vs, err := schema.Validate("capture", doc)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, v := range vs {
		problems = append(problems, v.String())
	}
	if len(problems) > 0 {
		return problems, nil
	}
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	imgPath := filepath.Join(filepath.Dir(path), filepath.FromSlash(r.Shot))
	info, err := content.ReadImageInfo(imgPath)
	if err != nil {
		return []string{fmt.Sprintf("shot %s: %v", r.Shot, err)}, nil
	}
	seen := map[string]bool{}
	for _, e := range r.Elements {
		if seen[e.Key] {
			problems = append(problems, fmt.Sprintf("element %q: duplicate key", e.Key))
		}
		seen[e.Key] = true
		if !content.BoxInside(e.Box[:], info) {
			problems = append(problems, fmt.Sprintf("element %q: box %v lies outside %s (%dx%d)", e.Key, e.Box, r.Shot, info.W, info.H))
		}
	}
	return problems, nil
}

func pngSize(data []byte) (int, int, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	return cfg.Width, cfg.Height, err
}

// RawElement is one element as measured in the page, in CSS pixels.
type RawElement struct {
	Key      string  `json:"key"`
	Selector string  `json:"selector"`
	Text     string  `json:"text"`
	Missing  bool    `json:"missing"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
}

// Raw is what a backend measured and shot, before export. The Playwright and
// pinchtab scripts send it to `muninn capture-finalize`, so every backend
// shares one implementation of crop, resize, box conversion and encoding.
type Raw struct {
	Backend    string       `json:"backend"`
	URL        string       `json:"url"`
	Viewport   Viewport     `json:"viewport"`
	DPR        float64      `json:"dpr"`
	MaxWidth   int          `json:"maxWidth"`
	Out        string       `json:"out"`        // output path without extension
	Screenshot string       `json:"screenshot"` // PNG file read by capture-finalize
	Elements   []RawElement `json:"elements"`
	Crop       *Rect        `json:"crop"`
}

// Finalize crops and scales the PNG screenshot, converts element boxes and
// writes <out>.webp and <out>.capture.json.
func Finalize(in Raw, pngData []byte) (*Result, error) {
	if in.DPR <= 0 {
		in.DPR = 1
	}
	w, h, err := pngSize(pngData)
	if err != nil {
		return nil, fmt.Errorf("screenshot must be PNG: %w", err)
	}
	t, area, err := NewTransform(in.DPR, w, h, in.Crop, in.MaxWidth)
	if err != nil {
		return nil, err
	}
	res := &Result{
		Backend: in.Backend, URL: in.URL, Viewport: in.Viewport,
		CapturedAt: time.Now().UTC().Format(time.RFC3339), Elements: []Measured{},
	}
	for _, e := range in.Elements {
		if e.Missing {
			return nil, fmt.Errorf("element %q: selector %q matches nothing", e.Key, e.Selector)
		}
		box, ok := t.Box(Rect{e.X, e.Y, e.W, e.H})
		if !ok {
			return nil, fmt.Errorf("element %q is not visible in the screenshot", e.Key)
		}
		res.Elements = append(res.Elements, Measured{Key: e.Key, Selector: e.Selector, Text: e.Text, Box: box})
	}
	img, err := Export(pngData, area, t)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(in.Out), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(in.Out+".webp", img, 0o644); err != nil {
		return nil, err
	}
	res.Shot = filepath.Base(in.Out) + ".webp"
	return res, WriteResult(in.Out, res)
}
