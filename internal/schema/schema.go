// Package schema validates content documents against the embedded JSON Schema.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	files "github.com/lamngockhuong/muninn/schema"
)

const (
	MainFile    = "muninn.schema.json"
	CaptureFile = "capture.schema.json"
	baseURL     = "https://github.com/lamngockhuong/muninn/schema/"
)

// Definitions that can be validated or printed on their own.
var Entities = []string{"config", "feature", "screen", "flow", "module", "glossary"}

var (
	compileOnce sync.Once
	compiled    map[string]*jsonschema.Schema
	compileErr  error
)

// Violation is one schema error at a location inside the document.
type Violation struct {
	Path    string // JSON pointer inside the document, "" for the root
	Message string
}

func (v Violation) String() string {
	if v.Path == "" {
		return v.Message
	}
	return v.Path + ": " + v.Message
}

func compileAll() {
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	for _, name := range []string{MainFile, CaptureFile} {
		raw, err := files.Files.ReadFile(name)
		if err != nil {
			compileErr = err
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			compileErr = fmt.Errorf("%s: %w", name, err)
			return
		}
		if err := c.AddResource(baseURL+name, doc); err != nil {
			compileErr = err
			return
		}
	}
	compiled = map[string]*jsonschema.Schema{}
	for _, e := range Entities {
		s, err := c.Compile(baseURL + MainFile + "#/$defs/" + e)
		if err != nil {
			compileErr = fmt.Errorf("compile %s: %w", e, err)
			return
		}
		compiled[e] = s
	}
	s, err := c.Compile(baseURL + CaptureFile)
	if err != nil {
		compileErr = fmt.Errorf("compile capture: %w", err)
		return
	}
	compiled["capture"] = s
}

// Validate checks a JSON-compatible value (maps, slices, strings, float64/int,
// bools, nil) against the named definition ("screen", "capture", ...).
func Validate(entity string, doc any) ([]Violation, error) {
	compileOnce.Do(compileAll)
	if compileErr != nil {
		return nil, compileErr
	}
	s, ok := compiled[entity]
	if !ok {
		return nil, fmt.Errorf("unknown schema entity %q", entity)
	}
	err := s.Validate(doc)
	if err == nil {
		return nil, nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return nil, err
	}
	var out []Violation
	collect(ve, &out)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return dedupe(out), nil
}

// collect keeps the leaf errors, which name the concrete problem, instead of
// the "doesn't validate with $ref" wrappers above them. A failed oneOf is
// reported once, with the description of the schema that declares it,
// because its leaf errors ("'not' failed") explain nothing.
func collect(ve *jsonschema.ValidationError, out *[]Violation) {
	if _, ok := ve.ErrorKind.(*kind.OneOf); ok {
		msg := "must match exactly one allowed shape"
		if d := description(strings.TrimSuffix(ve.SchemaURL, "/oneOf")); d != "" {
			msg += ": " + d
		}
		*out = append(*out, Violation{Path: pointer(ve.InstanceLocation), Message: msg})
		return
	}
	if len(ve.Causes) == 0 {
		*out = append(*out, Violation{Path: pointer(ve.InstanceLocation), Message: ve.ErrorKind.LocalizedString(printer)})
		return
	}
	for _, c := range ve.Causes {
		collect(c, out)
	}
}

// description returns the "description" of the schema at an absolute
// schema URL such as <base>muninn.schema.json#/$defs/flowStep.
func description(schemaURL string) string {
	file, ptr, ok := strings.Cut(strings.TrimPrefix(schemaURL, baseURL), "#")
	if !ok {
		return ""
	}
	raw, err := files.Files.ReadFile(file)
	if err != nil {
		return ""
	}
	var node any
	if json.Unmarshal(raw, &node) != nil {
		return ""
	}
	for _, seg := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
		m, ok := node.(map[string]any)
		if !ok {
			return ""
		}
		node = m[strings.ReplaceAll(strings.ReplaceAll(seg, "~1", "/"), "~0", "~")]
	}
	m, _ := node.(map[string]any)
	d, _ := m["description"].(string)
	return d
}

func dedupe(in []Violation) []Violation {
	seen := map[Violation]bool{}
	out := in[:0]
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func pointer(loc []string) string {
	if len(loc) == 0 {
		return ""
	}
	return "/" + strings.Join(loc, "/")
}

// Document returns a standalone schema for one entity (all definitions kept,
// root pointing at the entity), or the whole file when entity is "".
func Document(entity string) ([]byte, error) {
	if entity == "capture" {
		return files.Files.ReadFile(CaptureFile)
	}
	raw, err := files.Files.ReadFile(MainFile)
	if err != nil || entity == "" {
		return raw, err
	}
	known := false
	for _, e := range Entities {
		known = known || e == entity
	}
	if !known {
		return nil, fmt.Errorf("unknown entity %q (want one of %s, capture)", entity, strings.Join(Entities, ", "))
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	defs := doc["$defs"].(map[string]any)
	keep := map[string]any{}
	reachable(defs, entity, keep)
	out := map[string]any{
		"$schema":     doc["$schema"],
		"$id":         doc["$id"],
		"title":       defs[entity].(map[string]any)["title"],
		"description": doc["description"],
		"$ref":        "#/$defs/" + entity,
		"$defs":       keep,
	}
	return json.MarshalIndent(out, "", "  ")
}

// reachable copies a definition and every definition it references.
func reachable(defs map[string]any, name string, keep map[string]any) {
	if _, done := keep[name]; done {
		return
	}
	keep[name] = defs[name]
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if ref, ok := x["$ref"].(string); ok {
				if dep, ok := strings.CutPrefix(ref, "#/$defs/"); ok {
					reachable(defs, dep, keep)
				}
			}
			for _, c := range x {
				walk(c)
			}
		case []any:
			for _, c := range x {
				walk(c)
			}
		}
	}
	walk(defs[name])
}
