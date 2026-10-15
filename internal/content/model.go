package content

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// Kind is an entity type with its own directory and review state.
type Kind string

const (
	KindFeature Kind = "feature"
	KindScreen  Kind = "screen"
	KindFlow    Kind = "flow"
	KindModule  Kind = "module"
)

// Kinds in display and load order.
var Kinds = []Kind{KindFeature, KindScreen, KindFlow, KindModule}

// ParseKind accepts singular or plural names ("screen", "screens").
func ParseKind(s string) (Kind, bool) {
	for _, k := range Kinds {
		if s == string(k) || s == string(k)+"s" {
			return k, true
		}
	}
	return "", false
}

// Dir is the directory under the content root holding this kind.
func (k Kind) Dir() string { return string(k) + "s" }

// Review states written in files.
const (
	ReviewDraft            = "draft"
	ReviewApproved         = "approved"
	ReviewChangesRequested = "changes_requested"
)

type Localized map[string]string

type Review struct {
	Status       string            `yaml:"status" json:"status"`
	By           string            `yaml:"by,omitempty" json:"by,omitempty"`
	At           string            `yaml:"at,omitempty" json:"at,omitempty"`
	Note         string            `yaml:"note,omitempty" json:"note,omitempty"`
	ContentHash  string            `yaml:"contentHash,omitempty" json:"contentHash,omitempty"`
	SourceHashes map[string]string `yaml:"sourceHashes,omitempty" json:"sourceHashes,omitempty"`
}

type Labelled struct {
	ID    string    `yaml:"id"`
	Label Localized `yaml:"label"`
}

type Config struct {
	Muninn  int `yaml:"muninn"`
	Project struct {
		Name    string    `yaml:"name"`
		Summary Localized `yaml:"summary"`
	} `yaml:"project"`
	Languages struct {
		Default   string   `yaml:"default"`
		Available []string `yaml:"available"`
	} `yaml:"languages"`
	Repo struct {
		Root   string `yaml:"root"`
		URL    string `yaml:"url"`
		Branch string `yaml:"branch"`
		Path   string `yaml:"path"`
	} `yaml:"repo"`
	Roles  []Labelled          `yaml:"roles"`
	Areas  []Labelled          `yaml:"areas"`
	Owners map[string][]string `yaml:"owners"`
}

// Fields shared by every entity kind and needed by the loader and checks.
type common struct {
	ID      string   `yaml:"id"`
	Sources []string `yaml:"sources"`
	Review  Review   `yaml:"review"`
}

type Feature struct {
	common  `yaml:",inline"`
	Area    string   `yaml:"area"`
	Roles   []string `yaml:"roles"`
	Screens []string `yaml:"screens"`
	Flows   []string `yaml:"flows"`
	Modules []string `yaml:"modules"`
	Terms   []string `yaml:"terms"`
}

type Hotspot struct {
	N   int    `yaml:"n"`
	Box []int  `yaml:"box"`
	To  string `yaml:"to"`
}

type Variant struct {
	ID       string    `yaml:"id"`
	Shot     string    `yaml:"shot"`
	Hotspots []Hotspot `yaml:"hotspots"`
}

type Screen struct {
	common   `yaml:",inline"`
	Feature  string    `yaml:"feature"`
	Roles    []string  `yaml:"roles"`
	Variants []Variant `yaml:"variants"`
	Modules  []string  `yaml:"modules"`
	Links    []struct {
		To string `yaml:"to"`
	} `yaml:"links"`
}

type Step struct {
	N       int    `yaml:"n"`
	Action  string `yaml:"action"`
	Screen  string `yaml:"screen"`
	Variant string `yaml:"variant"`
	Shot    string `yaml:"shot"`
	Box     []int  `yaml:"box"`
}

type Flow struct {
	common  `yaml:",inline"`
	Feature string `yaml:"feature"`
	Role    string `yaml:"role"`
	Start   string `yaml:"start"`
	End     string `yaml:"end"`
	Steps   []Step `yaml:"steps"`
}

type Module struct {
	common    `yaml:",inline"`
	DependsOn []string `yaml:"dependsOn"`
	Endpoints []struct {
		Roles []string `yaml:"roles"`
	} `yaml:"endpoints"`
}

type Term struct {
	ID      string   `yaml:"id"`
	Related []string `yaml:"related"`
}

// Entity is one loaded feature, screen, flow or module.
type Entity struct {
	Kind Kind
	ID   string
	// Paths are relative to the content directory, with forward slashes.
	File         string            // main file
	Dir          string            // directory that shot paths are relative to
	Translations map[string]string // language -> translation file
	Bodies       map[string]string // language -> Markdown body
	Front        map[string]any    // frontmatter as JSON-compatible data
	Doc          *Document         // nil when the file could not be parsed
	Review       Review
	Sources      []string
	// Typed views used by cross-checks; nil when decoding failed.
	Feature *Feature
	Screen  *Screen
	Flow    *Flow
	Module  *Module
	// Images referenced by this entity, relative to the content directory.
	Images []string
	// ModTime of the main file, used for the stale draft warning.
	ModUnix int64
	// Schema or decode errors make cross-checks skip the entity.
	Broken bool
}

// Key is the stable "kind/id" reference used in the API and CLI.
func (e *Entity) Key() string { return string(e.Kind) + "/" + e.ID }

// RefKey builds a "kind/id" key.
func RefKey(k Kind, id string) string { return string(k) + "/" + id }

func decodeTyped(e *Entity, n *yaml.Node) error {
	var c *common
	switch e.Kind {
	case KindFeature:
		e.Feature = &Feature{}
		if err := n.Decode(e.Feature); err != nil {
			e.Feature = nil
			return err
		}
		c = &e.Feature.common
	case KindScreen:
		e.Screen = &Screen{}
		if err := n.Decode(e.Screen); err != nil {
			e.Screen = nil
			return err
		}
		c = &e.Screen.common
	case KindFlow:
		e.Flow = &Flow{}
		if err := n.Decode(e.Flow); err != nil {
			e.Flow = nil
			return err
		}
		c = &e.Flow.common
	case KindModule:
		e.Module = &Module{}
		if err := n.Decode(e.Module); err != nil {
			e.Module = nil
			return err
		}
		c = &e.Module.common
	}
	e.Review, e.Sources = c.Review, c.Sources
	return nil
}

// Pick returns the text for lang, falling back to fallback, then to any
// language in a stable order.
func Pick(v any, lang, fallback string) string {
	m, ok := v.(map[string]any)
	if !ok {
		s, _ := v.(string)
		return s
	}
	for _, l := range []string{lang, fallback} {
		if s, ok := m[l].(string); ok && s != "" {
			return s
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// Title is the entity title in lang, or its id when it has none.
func (e *Entity) Title(lang, fallback string) string {
	if t := Pick(e.Front["title"], lang, fallback); t != "" {
		return t
	}
	return e.ID
}
