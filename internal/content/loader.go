package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/lamngockhuong/muninn/internal/schema"
)

const (
	ConfigFile   = "muninn.yaml"
	GlossaryFile = "glossary.yaml"
)

// Severity of a validation issue.
type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

// Issue is one validation finding.
type Issue struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file"`             // relative to the content directory
	Entity   string   `json:"entity,omitempty"` // kind/id
	Path     string   `json:"path,omitempty"`   // location inside the document
	Message  string   `json:"message"`
}

func (i Issue) String() string {
	loc := i.File
	if i.Path != "" {
		loc += " " + i.Path
	}
	return fmt.Sprintf("%s: %s: %s", i.Severity, loc, i.Message)
}

// Ref is an incoming reference: entity From points at the key via Field.
type Ref struct {
	Kind  Kind   `json:"kind"`
	ID    string `json:"id"`
	Field string `json:"field"`
}

// Index is the in-memory view of a content directory.
type Index struct {
	Dir      string // absolute content directory
	RepoRoot string // absolute repository root used for sources
	Config   *Config
	// ConfigData is muninn.yaml as JSON-compatible data.
	ConfigData map[string]any
	Glossary   []Term
	// GlossaryData is glossary.yaml as JSON-compatible data.
	GlossaryData []any
	Entities     map[Kind]map[string]*Entity
	// RefsTo maps a "kind/id" (or "term/id") key to its incoming references.
	RefsTo map[string][]Ref
	Images map[string]ImageInfo
	Issues []Issue
}

// Get returns an entity by kind and id.
func (ix *Index) Get(k Kind, id string) *Entity { return ix.Entities[k][id] }

// All returns every entity ordered by kind then id.
func (ix *Index) All() []*Entity {
	var out []*Entity
	for _, k := range Kinds {
		ids := make([]string, 0, len(ix.Entities[k]))
		for id := range ix.Entities[k] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			out = append(out, ix.Entities[k][id])
		}
	}
	return out
}

// IssuesFor returns the issues attached to one entity.
func (ix *Index) IssuesFor(key string) []Issue {
	var out []Issue
	for _, is := range ix.Issues {
		if is.Entity == key {
			out = append(out, is)
		}
	}
	return out
}

// ErrorCount counts issues of error severity.
func (ix *Index) ErrorCount() int {
	n := 0
	for _, is := range ix.Issues {
		if is.Severity == SevError {
			n++
		}
	}
	return n
}

// Abs resolves a content-relative path to an absolute path.
func (ix *Index) Abs(rel string) string { return filepath.Join(ix.Dir, filepath.FromSlash(rel)) }

// Load reads a content directory, validates every file against the schema,
// cross-checks references, images and sources, and builds the index. It only
// returns an error when the directory or muninn.yaml cannot be read at all;
// content problems are reported as Issues.
func Load(dir string) (*Index, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("content directory %s not found", dir)
	}
	ix := &Index{
		Dir:      abs,
		Entities: map[Kind]map[string]*Entity{},
		RefsTo:   map[string][]Ref{},
		Images:   map[string]ImageInfo{},
	}
	for _, k := range Kinds {
		ix.Entities[k] = map[string]*Entity{}
	}
	if err := ix.loadConfig(); err != nil {
		return nil, err
	}
	ix.loadGlossary()
	ix.loadEntityDirs(KindFeature)
	ix.loadEntityDirs(KindScreen)
	ix.loadEntityDirs(KindFlow)
	ix.loadModules()
	ix.check()
	sort.SliceStable(ix.Issues, func(i, j int) bool {
		a, b := ix.Issues[i], ix.Issues[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Path < b.Path
	})
	return ix, nil
}

func (ix *Index) addIssue(sev Severity, file, entity, p, format string, args ...any) {
	ix.Issues = append(ix.Issues, Issue{Severity: sev, File: file, Entity: entity, Path: p, Message: fmt.Sprintf(format, args...)})
}

func (ix *Index) schemaCheck(def string, data any, file, entity string) bool {
	vs, err := schema.Validate(def, data)
	if err != nil {
		ix.addIssue(SevError, file, entity, "", "schema: %v", err)
		return false
	}
	for _, v := range vs {
		ix.addIssue(SevError, file, entity, v.Path, "%s", v.Message)
	}
	return len(vs) == 0
}

func (ix *Index) loadConfig() error {
	raw, err := os.ReadFile(filepath.Join(ix.Dir, ConfigFile))
	if err != nil {
		return fmt.Errorf("%s not found in %s (run `muninn init` to create one)", ConfigFile, ix.Dir)
	}
	ix.Config = &Config{}
	ix.Config.Repo.Root = ".."
	node, err := ParseYAML(raw)
	if err != nil {
		ix.addIssue(SevError, ConfigFile, "", "", "%v", err)
		ix.RepoRoot = filepath.Dir(ix.Dir)
		return nil
	}
	data, err := Value(node)
	if err != nil {
		ix.addIssue(SevError, ConfigFile, "", "", "%v", err)
	} else {
		ix.ConfigData, _ = data.(map[string]any)
		if ix.schemaCheck("config", data, ConfigFile, "") {
			if err := node.Decode(ix.Config); err != nil {
				ix.addIssue(SevError, ConfigFile, "", "", "%v", err)
			}
		}
	}
	if ix.Config.Repo.Root == "" {
		ix.Config.Repo.Root = ".."
	}
	ix.RepoRoot = filepath.Clean(filepath.Join(ix.Dir, filepath.FromSlash(ix.Config.Repo.Root)))
	return nil
}

func (ix *Index) loadGlossary() {
	raw, err := os.ReadFile(filepath.Join(ix.Dir, GlossaryFile))
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		ix.addIssue(SevError, GlossaryFile, "", "", "%v", err)
		return
	}
	node, err := ParseYAML(raw)
	if err != nil {
		ix.addIssue(SevError, GlossaryFile, "", "", "%v", err)
		return
	}
	data, err := Value(node)
	if err != nil {
		ix.addIssue(SevError, GlossaryFile, "", "", "%v", err)
		return
	}
	if data == nil {
		return
	}
	if !ix.schemaCheck("glossary", data, GlossaryFile, "") {
		return
	}
	ix.GlossaryData, _ = data.([]any)
	if err := node.Decode(&ix.Glossary); err != nil {
		ix.addIssue(SevError, GlossaryFile, "", "", "%v", err)
	}
}

// loadEntityDirs loads features/<id>/feature.md style directories.
func (ix *Index) loadEntityDirs(k Kind) {
	root := filepath.Join(ix.Dir, k.Dir())
	dirs, err := os.ReadDir(root)
	if err != nil {
		return
	}
	base := string(k)
	for _, d := range dirs {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		relDir := k.Dir() + "/" + d.Name()
		files, _ := os.ReadDir(filepath.Join(root, d.Name()))
		translations := map[string]string{}
		hasMain := false
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if name == base+".md" {
				hasMain = true
			} else if lang, ok := translationLang(name, base); ok {
				translations[lang] = relDir + "/" + name
			}
		}
		if !hasMain {
			for _, t := range translations {
				ix.addIssue(SevError, t, "", "", "translation has no main file %s/%s.md", relDir, base)
			}
			if len(translations) == 0 {
				ix.addIssue(SevWarning, relDir, "", "", "directory has no %s.md and is ignored", base)
			}
			continue
		}
		ix.loadEntity(k, d.Name(), relDir+"/"+base+".md", relDir, translations)
	}
}

// loadModules loads modules/<id>.md and modules/<id>.<lang>.md.
func (ix *Index) loadModules() {
	root := filepath.Join(ix.Dir, KindModule.Dir())
	files, err := os.ReadDir(root)
	if err != nil {
		return
	}
	mains := map[string]bool{}
	translations := map[string]map[string]string{}
	for _, f := range files {
		name := f.Name()
		if f.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		stem := strings.TrimSuffix(name, ".md")
		if i := strings.LastIndexByte(stem, '.'); i >= 0 {
			id, lang := stem[:i], stem[i+1:]
			if translations[id] == nil {
				translations[id] = map[string]string{}
			}
			translations[id][lang] = "modules/" + name
			continue
		}
		mains[stem] = true
	}
	for id, ts := range translations {
		if !mains[id] {
			for _, t := range ts {
				ix.addIssue(SevError, t, "", "", "translation has no main file modules/%s.md", id)
			}
		}
	}
	ids := make([]string, 0, len(mains))
	for id := range mains {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ts := translations[id]
		if ts == nil {
			ts = map[string]string{}
		}
		ix.loadEntity(KindModule, id, "modules/"+id+".md", "modules", ts)
	}
}

func translationLang(name, base string) (string, bool) {
	if !strings.HasPrefix(name, base+".") || !strings.HasSuffix(name, ".md") {
		return "", false
	}
	lang := strings.TrimSuffix(strings.TrimPrefix(name, base+"."), ".md")
	return lang, lang != "" && !strings.Contains(lang, ".")
}

func (ix *Index) loadEntity(k Kind, name, file, dir string, translations map[string]string) {
	e := &Entity{Kind: k, ID: name, File: file, Dir: dir, Translations: translations, Bodies: map[string]string{}}
	key := e.Key()
	if st, err := os.Stat(ix.Abs(file)); err == nil {
		e.ModUnix = st.ModTime().Unix()
	}
	ix.Entities[k][name] = e
	src, err := os.ReadFile(ix.Abs(file))
	if err != nil {
		e.Broken = true
		ix.addIssue(SevError, file, key, "", "%v", err)
		return
	}
	doc, err := ParseDocument(src)
	if err != nil {
		e.Broken = true
		ix.addIssue(SevError, file, key, "", "%v", err)
		return
	}
	e.Doc = doc
	e.Bodies[ix.defaultLang()] = doc.Body()
	for lang, t := range translations {
		b, err := os.ReadFile(ix.Abs(t))
		if err != nil {
			ix.addIssue(SevError, t, key, "", "%v", err)
			continue
		}
		if HasFrontmatter(b) {
			ix.addIssue(SevError, t, key, "", "translation files contain a Markdown body only, no frontmatter")
			continue
		}
		if !ix.languageAvailable(lang) {
			ix.addIssue(SevWarning, t, key, "", "language %q is not listed in languages.available", lang)
		}
		e.Bodies[lang] = string(b)
	}
	data, err := Value(doc.Node)
	if err != nil {
		e.Broken = true
		ix.addIssue(SevError, file, key, "", "%v", err)
		return
	}
	e.Front, _ = data.(map[string]any)
	if !ix.schemaCheck(string(k), data, file, key) {
		e.Broken = true
		// Still read the review block so the queue can show the entity.
		if rv, ok := e.Front["review"].(map[string]any); ok {
			e.Review.Status, _ = rv["status"].(string)
		}
		return
	}
	if err := decodeTyped(e, doc.Node); err != nil {
		e.Broken = true
		ix.addIssue(SevError, file, key, "", "%v", err)
		return
	}
	if id := frontID(doc.Node); id != name {
		ix.addIssue(SevError, file, key, "/id", "id %q must equal the %s name %q", id, map[bool]string{true: "file", false: "directory"}[k == KindModule], name)
	}
	e.Images = entityImages(e)
}

func frontID(n *yaml.Node) string {
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == "id" {
			return n.Content[i+1].Value
		}
	}
	return ""
}

func entityImages(e *Entity) []string {
	var shots []string
	switch e.Kind {
	case KindScreen:
		for _, v := range e.Screen.Variants {
			shots = append(shots, v.Shot)
		}
	case KindFlow:
		for _, s := range e.Flow.Steps {
			if s.Shot != "" {
				shots = append(shots, s.Shot)
			}
		}
	}
	out := make([]string, 0, len(shots))
	seen := map[string]bool{}
	for _, s := range shots {
		p := path.Join(e.Dir, s)
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func (ix *Index) defaultLang() string {
	if ix.Config != nil && ix.Config.Languages.Default != "" {
		return ix.Config.Languages.Default
	}
	return "en"
}

// DefaultLanguage is the language of the main .md bodies.
func (ix *Index) DefaultLanguage() string { return ix.defaultLang() }

func (ix *Index) languageAvailable(lang string) bool {
	if ix.Config == nil {
		return true
	}
	for _, l := range ix.Config.Languages.Available {
		if l == lang {
			return true
		}
	}
	return lang == ix.defaultLang()
}
