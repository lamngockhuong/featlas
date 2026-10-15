package server

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/review"
)

// Snapshot is one immutable build of the index with computed review states.
// API responses are derived from it; the static export writes the same data.
type Snapshot struct {
	Index      *content.Index
	Statuses   map[string]review.Status
	Generation int64
	// Revisions hold each entity's content hash (files without the review
	// block), so a review can be tied to the version the reviewer saw.
	Revisions map[string]string
}

// NewSnapshot loads the content directory and computes review states.
func NewSnapshot(dir string, h *review.Hasher, gen int64) (*Snapshot, error) {
	ix, err := content.Load(dir)
	if err != nil {
		return nil, err
	}
	if h == nil || h.Root() != ix.RepoRoot {
		h = review.NewHasher(ix.RepoRoot)
	}
	revs := map[string]string{}
	for _, e := range ix.All() {
		if hash, err := review.ContentHash(ix, e); err == nil {
			revs[e.Key()] = hash
		}
	}
	return &Snapshot{Index: ix, Statuses: review.ComputeAll(ix, h), Generation: gen, Revisions: revs}, nil
}

// Capabilities tell the client what this server allows.
type Capabilities struct {
	CanReview      bool   `json:"canReview"`
	NeedsToken     bool   `json:"needsToken"`
	ReadOnlyReason string `json:"readOnlyReason,omitempty"`
	Static         bool   `json:"static"`
	// LocalRepo is the absolute repository root, set only when the server
	// runs locally, for vscode:// links.
	LocalRepo string `json:"localRepo,omitempty"`
}

type ProjectResponse struct {
	Config       map[string]any `json:"config"`
	Counts       map[string]int `json:"counts"`
	Total        int            `json:"total"`
	Validation   ValidationSum  `json:"validation"`
	Capabilities Capabilities   `json:"capabilities"`
	Version      string         `json:"version"`
	Generation   int64          `json:"generation"`
}

type ValidationSum struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

func (s *Snapshot) validationSum() ValidationSum {
	e := s.Index.ErrorCount()
	return ValidationSum{Errors: e, Warnings: len(s.Index.Issues) - e}
}

func (s *Snapshot) Project(caps Capabilities, version string) ProjectResponse {
	counts := map[string]int{}
	for _, st := range review.States {
		counts[st] = 0
	}
	for _, st := range s.Statuses {
		counts[st.State]++
	}
	cfg := s.Index.ConfigData
	if cfg == nil {
		cfg = map[string]any{}
	}
	return ProjectResponse{
		Config: cfg, Counts: counts, Total: len(s.Statuses),
		Validation: s.validationSum(), Capabilities: caps, Version: version, Generation: s.Generation,
	}
}

// Summary is the list view of an entity.
type Summary struct {
	Key      string         `json:"key"`
	Kind     content.Kind   `json:"kind"`
	ID       string         `json:"id"`
	Title    any            `json:"title"`
	Summary  any            `json:"summary,omitempty"`
	Area     string         `json:"area,omitempty"`
	Feature  string         `json:"feature,omitempty"`
	Roles    []string       `json:"roles"`
	Owners   []string       `json:"owners"`
	Status   review.Status  `json:"status"`
	Review   content.Review `json:"review"`
	Thumb    *Media         `json:"thumb,omitempty"`
	Counts   map[string]int `json:"counts,omitempty"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	File     string         `json:"file"`
	Revision string         `json:"revision"`
}

// Media is an image with its real size, so the client can place hotspots
// as percentages before the image loads.
type Media struct {
	URL   string `json:"url"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Bytes int64  `json:"bytes"`
}

// MediaURL is the URL of a content-relative file.
func MediaURL(rel string) string {
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "/content/" + strings.Join(parts, "/")
}

func (s *Snapshot) media(rel string) *Media {
	info, ok := s.Index.Images[rel]
	if !ok {
		var err error
		if info, err = content.ReadImageInfo(s.Index.Abs(rel)); err != nil {
			return nil
		}
	}
	return &Media{URL: MediaURL(rel), W: info.W, H: info.H, Bytes: info.Bytes}
}

func strs(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func str(v any) string { s, _ := v.(string); return s }

// featureOf returns the feature id an entity belongs to.
func featureOf(e *content.Entity) string {
	if e.Kind == content.KindFeature {
		return e.ID
	}
	return str(e.Front["feature"])
}

func (s *Snapshot) areaOf(e *content.Entity) string {
	if f := s.Index.Get(content.KindFeature, featureOf(e)); f != nil {
		return str(f.Front["area"])
	}
	return ""
}

func (s *Snapshot) rolesOf(e *content.Entity) []string {
	if e.Kind == content.KindFlow {
		if r := str(e.Front["role"]); r != "" {
			return []string{r}
		}
	}
	roles := strs(e.Front["roles"])
	if len(roles) == 0 && e.Kind != content.KindFeature {
		if f := s.Index.Get(content.KindFeature, featureOf(e)); f != nil {
			roles = strs(f.Front["roles"])
		}
	}
	return roles
}

func (s *Snapshot) Summary(e *content.Entity) Summary {
	sum := Summary{
		Key: e.Key(), Kind: e.Kind, ID: e.ID, Title: e.Front["title"], Summary: e.Front["summary"],
		Area: s.areaOf(e), Feature: featureOf(e), Roles: s.rolesOf(e), Owners: []string{},
		Status: s.Statuses[e.Key()], Review: e.Review, File: e.File, Revision: s.Revisions[e.Key()],
	}
	if sum.Title == nil {
		sum.Title = map[string]any{s.Index.DefaultLanguage(): e.ID}
	}
	if s.Index.Config != nil && sum.Area != "" {
		if o := s.Index.Config.Owners[sum.Area]; o != nil {
			sum.Owners = o
		}
	}
	for _, is := range s.Index.IssuesFor(e.Key()) {
		if is.Severity == content.SevError {
			sum.Errors++
		} else {
			sum.Warnings++
		}
	}
	switch {
	case e.Kind == content.KindScreen && e.Screen != nil && len(e.Screen.Variants) > 0:
		sum.Thumb = s.media(path.Join(e.Dir, e.Screen.Variants[0].Shot))
	case e.Kind == content.KindFeature:
		sum.Counts = map[string]int{"screens": len(strs(e.Front["screens"])), "flows": len(strs(e.Front["flows"])), "modules": len(strs(e.Front["modules"]))}
		if sc := strs(e.Front["screens"]); len(sc) > 0 {
			if first := s.Index.Get(content.KindScreen, sc[0]); first != nil && first.Screen != nil && len(first.Screen.Variants) > 0 {
				sum.Thumb = s.media(path.Join(first.Dir, first.Screen.Variants[0].Shot))
			}
		}
	}
	return sum
}

// Filter selects entities for /api/entities.
type Filter struct{ Kind, State, Role, Area string }

func (s *Snapshot) Summaries(f Filter) []Summary {
	out := []Summary{}
	for _, e := range s.Index.All() {
		if f.Kind != "" {
			if k, ok := content.ParseKind(f.Kind); !ok || k != e.Kind {
				continue
			}
		}
		sum := s.Summary(e)
		if f.State != "" && sum.Status.State != f.State {
			continue
		}
		if f.Area != "" && sum.Area != f.Area {
			continue
		}
		if f.Role != "" && !contains(sum.Roles, f.Role) {
			continue
		}
		out = append(out, sum)
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// StepMedia is the image of one flow step, resolved from its screen variant
// or its own shot.
type StepMedia struct {
	N     int    `json:"n"`
	Media *Media `json:"media,omitempty"`
}

// Detail is the full view of one entity.
type Detail struct {
	Summary
	Front        map[string]any    `json:"front"`
	Bodies       map[string]string `json:"bodies"`
	Translations []string          `json:"translations"`
	RefsTo       []content.Ref     `json:"refsTo"`
	Issues       []content.Issue   `json:"issues"`
	Media        map[string]*Media `json:"media"`
	Steps        []StepMedia       `json:"steps,omitempty"`
	Sources      []Source          `json:"sources"`
}

// Source is a source path with its current state relative to the review.
type Source struct {
	Path   string `json:"path"`
	Dir    bool   `json:"dir"`
	Change string `json:"change,omitempty"` // source-changed | source-missing | source-added
}

func (s *Snapshot) Detail(e *content.Entity) Detail {
	d := Detail{
		Summary: s.Summary(e), Front: e.Front, Bodies: e.Bodies, Translations: []string{},
		RefsTo: s.Index.RefsTo[e.Key()], Issues: s.Index.IssuesFor(e.Key()),
		Media: map[string]*Media{}, Sources: []Source{},
	}
	if d.Front == nil {
		d.Front = map[string]any{}
	}
	for l := range e.Translations {
		d.Translations = append(d.Translations, l)
	}
	sort.Strings(d.Translations)
	if d.RefsTo == nil {
		d.RefsTo = []content.Ref{}
	}
	if d.Issues == nil {
		d.Issues = []content.Issue{}
	}
	for _, img := range e.Images {
		if m := s.media(img); m != nil {
			d.Media[strings.TrimPrefix(img, e.Dir+"/")] = m
		}
	}
	if e.Flow != nil {
		for _, st := range e.Flow.Steps {
			sm := StepMedia{N: st.N}
			switch {
			case st.Shot != "":
				sm.Media = s.media(path.Join(e.Dir, st.Shot))
			case st.Screen != "":
				if scr := s.Index.Get(content.KindScreen, st.Screen); scr != nil && scr.Screen != nil {
					for i, v := range scr.Screen.Variants {
						if v.ID == st.Variant || (st.Variant == "" && i == 0) {
							sm.Media = s.media(path.Join(scr.Dir, v.Shot))
						}
					}
				}
			}
			d.Steps = append(d.Steps, sm)
		}
	}
	changes := map[string]string{}
	for _, r := range d.Status.Reasons {
		if r.Path != "" {
			changes[r.Path] = r.Kind
		}
	}
	for _, src := range e.Sources {
		d.Sources = append(d.Sources, Source{Path: src, Dir: strings.HasSuffix(src, "/"), Change: changes[src]})
	}
	return d
}

// SearchDoc is one document of the client-side search index.
type SearchDoc struct {
	Key   string            `json:"key"`
	Kind  string            `json:"kind"`
	ID    string            `json:"id"`
	Title map[string]string `json:"title"`
	Text  string            `json:"text"`
	Route string            `json:"route,omitempty"`
}

func (s *Snapshot) SearchIndex() []SearchDoc {
	docs := []SearchDoc{}
	for _, e := range s.Index.All() {
		var b strings.Builder
		collectText(e.Front, &b, map[string]bool{"review": true, "sources": true, "id": true, "box": true, "shot": true, "captured": true})
		for _, body := range e.Bodies {
			b.WriteString(body)
			b.WriteByte('\n')
		}
		docs = append(docs, SearchDoc{Key: e.Key(), Kind: string(e.Kind), ID: e.ID, Title: localized(e.Front["title"], e.ID), Text: b.String(), Route: str(e.Front["route"])})
	}
	for _, t := range s.Index.GlossaryData {
		m, _ := t.(map[string]any)
		id := str(m["id"])
		var b strings.Builder
		collectText(m["meaning"], &b, nil)
		docs = append(docs, SearchDoc{Key: "term/" + id, Kind: "term", ID: id, Title: localized(m["term"], id), Text: b.String()})
	}
	return docs
}

func localized(v any, fallback string) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]any); ok {
		for k, x := range m {
			if s, ok := x.(string); ok {
				out[k] = s
			}
		}
	}
	if len(out) == 0 {
		out["en"] = fallback
	}
	return out
}

// collectText gathers every string value except under skipped keys.
func collectText(v any, b *strings.Builder, skip map[string]bool) {
	switch x := v.(type) {
	case string:
		b.WriteString(x)
		b.WriteByte('\n')
	case []any:
		for _, i := range x {
			collectText(i, b, skip)
		}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !skip[k] {
				collectText(x[k], b, skip)
			}
		}
	}
}

type GraphNode struct {
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title any    `json:"title"`
	State string `json:"state"`
}

type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Field string `json:"field"`
}

type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

func (s *Snapshot) Graph() Graph {
	g := Graph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	for _, e := range s.Index.All() {
		g.Nodes = append(g.Nodes, GraphNode{Key: e.Key(), Kind: string(e.Kind), ID: e.ID, Title: e.Front["title"], State: s.Statuses[e.Key()].State})
	}
	keys := make([]string, 0, len(s.Index.RefsTo))
	for k := range s.Index.RefsTo {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := map[GraphEdge]bool{}
	for _, to := range keys {
		if strings.HasPrefix(to, "term/") {
			continue
		}
		for _, r := range s.Index.RefsTo[to] {
			if r.Kind == "term" {
				continue
			}
			// One edge per entity pair; the first field wins.
			e := GraphEdge{From: content.RefKey(r.Kind, r.ID), To: to}
			if seen[e] {
				continue
			}
			seen[e] = true
			e.Field = r.Field
			g.Edges = append(g.Edges, e)
		}
	}
	return g
}

type ValidationResponse struct {
	ValidationSum
	Issues []content.Issue `json:"issues"`
}

func (s *Snapshot) Validation() ValidationResponse {
	issues := s.Index.Issues
	if issues == nil {
		issues = []content.Issue{}
	}
	return ValidationResponse{ValidationSum: s.validationSum(), Issues: issues}
}

// Glossary returns glossary.yaml with incoming references per term.
func (s *Snapshot) Glossary() []map[string]any {
	out := []map[string]any{}
	for _, t := range s.Index.GlossaryData {
		m, _ := t.(map[string]any)
		cp := map[string]any{}
		for k, v := range m {
			cp[k] = v
		}
		refs := s.Index.RefsTo["term/"+str(m["id"])]
		if refs == nil {
			refs = []content.Ref{}
		}
		cp["refsTo"] = refs
		out = append(out, cp)
	}
	return out
}
