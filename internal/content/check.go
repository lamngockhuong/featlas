package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// LargeImageBytes triggers a warning: screenshots should be WebP, at most 1440px wide.
	LargeImageBytes = 400 * 1024
	// StaleDraftAge triggers a warning for drafts nobody reviewed.
	StaleDraftAge = 30 * 24 * time.Hour
)

// refKind names what an id field points at.
type refKind string

const (
	refFeature refKind = "feature"
	refScreen  refKind = "screen"
	refFlow    refKind = "flow"
	refModule  refKind = "module"
	refTerm    refKind = "term"
	refRole    refKind = "role"
	refArea    refKind = "area"
)

// check runs every cross-file rule after all files are loaded.
func (ix *Index) check() {
	ids := ix.knownIDs()
	ix.checkConfig(ids)
	ix.checkGlossary(ids)
	ix.checkDuplicateIDs()
	now := time.Now()
	for _, e := range ix.All() {
		if e.Broken {
			continue
		}
		c := checker{ix: ix, e: e, ids: ids}
		switch e.Kind {
		case KindFeature:
			c.checkFeature()
		case KindScreen:
			c.checkScreen()
		case KindFlow:
			c.checkFlow()
		case KindModule:
			c.checkModule()
		}
		c.checkSources()
		if e.Review.Status == ReviewDraft && e.ModUnix > 0 && now.Sub(time.Unix(e.ModUnix, 0)) > StaleDraftAge {
			ix.addIssue(SevWarning, e.File, e.Key(), "/review", "draft has not been reviewed for more than 30 days")
		}
	}
}

func (ix *Index) knownIDs() map[refKind]map[string]bool {
	ids := map[refKind]map[string]bool{}
	add := func(k refKind, id string) {
		if ids[k] == nil {
			ids[k] = map[string]bool{}
		}
		ids[k][id] = true
	}
	for _, k := range Kinds {
		for id := range ix.Entities[k] {
			add(refKind(k), id)
		}
	}
	for _, t := range ix.Glossary {
		add(refTerm, t.ID)
	}
	if ix.Config != nil {
		for _, r := range ix.Config.Roles {
			add(refRole, r.ID)
		}
		for _, a := range ix.Config.Areas {
			add(refArea, a.ID)
		}
	}
	return ids
}

func (ix *Index) checkConfig(ids map[refKind]map[string]bool) {
	if ix.Config == nil {
		return
	}
	for area := range ix.Config.Owners {
		if !ids[refArea][area] {
			ix.addIssue(SevError, ConfigFile, "", "/owners/"+area, "unknown area %q", area)
		}
	}
	if len(ix.Config.Languages.Available) > 0 && !ix.languageAvailableList(ix.Config.Languages.Default) {
		ix.addIssue(SevWarning, ConfigFile, "", "/languages/available", "should include the default language %q", ix.Config.Languages.Default)
	}
	if st, err := os.Stat(ix.RepoRoot); err != nil || !st.IsDir() {
		ix.addIssue(SevError, ConfigFile, "", "/repo/root", "repository root %s does not exist", ix.RepoRoot)
	}
	seen := map[string]bool{}
	for i, r := range ix.Config.Roles {
		if seen["role/"+r.ID] {
			ix.addIssue(SevError, ConfigFile, "", fmt.Sprintf("/roles/%d", i), "duplicate role id %q", r.ID)
		}
		seen["role/"+r.ID] = true
	}
	for i, a := range ix.Config.Areas {
		if seen["area/"+a.ID] {
			ix.addIssue(SevError, ConfigFile, "", fmt.Sprintf("/areas/%d", i), "duplicate area id %q", a.ID)
		}
		seen["area/"+a.ID] = true
	}
}

func (ix *Index) languageAvailableList(lang string) bool {
	for _, l := range ix.Config.Languages.Available {
		if l == lang {
			return true
		}
	}
	return false
}

func (ix *Index) checkGlossary(ids map[refKind]map[string]bool) {
	seen := map[string]bool{}
	for i, t := range ix.Glossary {
		p := fmt.Sprintf("/%d", i)
		if seen[t.ID] {
			ix.addIssue(SevError, GlossaryFile, "", p+"/id", "duplicate id %q", t.ID)
		}
		seen[t.ID] = true
		for j, r := range t.Related {
			if !ids[refTerm][r] {
				ix.addIssue(SevError, GlossaryFile, "", fmt.Sprintf("%s/related/%d", p, j), "unknown term %q", r)
			} else {
				ix.RefsTo["term/"+r] = append(ix.RefsTo["term/"+r], Ref{Kind: "term", ID: t.ID, Field: "related"})
			}
		}
	}
}

// checkDuplicateIDs reports two entities of one kind declaring the same id.
// The id-equals-directory rule already flags each; this names the clash.
func (ix *Index) checkDuplicateIDs() {
	for _, k := range Kinds {
		owners := map[string][]*Entity{}
		for _, e := range ix.Entities[k] {
			if id, _ := e.Front["id"].(string); id != "" {
				owners[id] = append(owners[id], e)
			}
		}
		for id, es := range owners {
			if len(es) < 2 {
				continue
			}
			files := make([]string, len(es))
			for i, e := range es {
				files[i] = e.File
			}
			for _, e := range es {
				ix.addIssue(SevError, e.File, e.Key(), "/id", "duplicate %s id %q (declared in %s)", k, id, strings.Join(files, ", "))
			}
		}
	}
}

type checker struct {
	ix  *Index
	e   *Entity
	ids map[refKind]map[string]bool
}

func (c *checker) errorf(p, format string, args ...any) {
	c.ix.addIssue(SevError, c.e.File, c.e.Key(), p, format, args...)
}

// ref checks one id reference and records the incoming edge.
func (c *checker) ref(k refKind, id, p, field string) {
	if id == "" {
		return
	}
	if !c.ids[k][id] {
		c.errorf(p, "unknown %s %q", k, id)
		return
	}
	if k == refRole || k == refArea {
		return
	}
	key := string(k) + "/" + id
	for _, r := range c.ix.RefsTo[key] {
		if r.Kind == c.e.Kind && r.ID == c.e.ID && r.Field == field {
			return
		}
	}
	c.ix.RefsTo[key] = append(c.ix.RefsTo[key], Ref{Kind: c.e.Kind, ID: c.e.ID, Field: field})
}

func (c *checker) refs(k refKind, ids []string, p, field string) {
	for i, id := range ids {
		c.ref(k, id, fmt.Sprintf("%s/%d", p, i), field)
	}
}

// image checks that an image exists and returns its real size.
func (c *checker) image(rel, p string) (ImageInfo, bool) {
	if info, ok := c.ix.Images[rel]; ok {
		return info, true
	}
	info, err := ReadImageInfo(c.ix.Abs(rel))
	if os.IsNotExist(err) {
		c.errorf(p, "image %s not found", rel)
		return info, false
	}
	if err != nil {
		c.errorf(p, "image %s cannot be read: %v", rel, err)
		return info, false
	}
	c.ix.Images[rel] = info
	if info.Bytes > LargeImageBytes {
		c.ix.addIssue(SevWarning, c.e.File, c.e.Key(), p, "image %s is %d KB; keep screenshots under 400 KB (WebP, max 1440px wide)", rel, info.Bytes/1024)
	}
	return info, true
}

func (c *checker) box(box []int, img ImageInfo, imgRel, p string) {
	if !BoxInside(box, img) {
		c.errorf(p, "box %v lies outside %s (%dx%d)", box, imgRel, img.W, img.H)
	}
}

func (c *checker) checkFeature() {
	f := c.e.Feature
	c.ref(refArea, f.Area, "/area", "area")
	c.refs(refRole, f.Roles, "/roles", "roles")
	c.refs(refScreen, f.Screens, "/screens", "screens")
	c.refs(refFlow, f.Flows, "/flows", "flows")
	c.refs(refModule, f.Modules, "/modules", "modules")
	c.refs(refTerm, f.Terms, "/terms", "terms")
}

func (c *checker) checkScreen() {
	s := c.e.Screen
	c.ref(refFeature, s.Feature, "/feature", "feature")
	c.refs(refRole, s.Roles, "/roles", "roles")
	c.refs(refModule, s.Modules, "/modules", "modules")
	for i, l := range s.Links {
		c.ref(refScreen, l.To, fmt.Sprintf("/links/%d/to", i), "links")
	}
	variants := map[string]bool{}
	for i, v := range s.Variants {
		vp := fmt.Sprintf("/variants/%d", i)
		if variants[v.ID] {
			c.errorf(vp+"/id", "duplicate variant id %q", v.ID)
		}
		variants[v.ID] = true
		rel := joinRel(c.e.Dir, v.Shot)
		img, ok := c.image(rel, vp+"/shot")
		numbers := map[int]bool{}
		for j, h := range v.Hotspots {
			hp := fmt.Sprintf("%s/hotspots/%d", vp, j)
			if numbers[h.N] {
				c.errorf(hp+"/n", "duplicate hotspot number %d", h.N)
			}
			numbers[h.N] = true
			c.ref(refScreen, h.To, hp+"/to", "hotspots")
			if ok {
				c.box(h.Box, img, rel, hp+"/box")
			}
		}
	}
}

func (c *checker) checkFlow() {
	f := c.e.Flow
	c.ref(refFeature, f.Feature, "/feature", "feature")
	c.ref(refRole, f.Role, "/role", "role")
	c.ref(refScreen, f.Start, "/start", "start")
	c.ref(refScreen, f.End, "/end", "end")
	for i, s := range f.Steps {
		sp := fmt.Sprintf("/steps/%d", i)
		if s.N != i+1 {
			c.errorf(sp+"/n", "step number %d out of order (want %d)", s.N, i+1)
		}
		var img ImageInfo
		var imgRel string
		ok := false
		switch {
		case s.Screen != "":
			c.ref(refScreen, s.Screen, sp+"/screen", "steps")
			img, imgRel, ok = c.variantImage(s.Screen, s.Variant, sp)
		case s.Shot != "":
			imgRel = joinRel(c.e.Dir, s.Shot)
			img, ok = c.image(imgRel, sp+"/shot")
		}
		if ok && len(s.Box) > 0 {
			c.box(s.Box, img, imgRel, sp+"/box")
		}
	}
}

// variantImage resolves the screenshot of a screen variant used by a step.
func (c *checker) variantImage(screenID, variantID, p string) (ImageInfo, string, bool) {
	scr := c.ix.Get(KindScreen, screenID)
	if scr == nil || scr.Screen == nil || len(scr.Screen.Variants) == 0 {
		return ImageInfo{}, "", false
	}
	v := scr.Screen.Variants[0]
	if variantID != "" {
		found := false
		for _, cand := range scr.Screen.Variants {
			if cand.ID == variantID {
				v, found = cand, true
			}
		}
		if !found {
			c.errorf(p+"/variant", "screen %q has no variant %q", screenID, variantID)
			return ImageInfo{}, "", false
		}
	}
	rel := joinRel(scr.Dir, v.Shot)
	info, err := ReadImageInfo(c.ix.Abs(rel))
	if err != nil {
		return ImageInfo{}, "", false // reported on the screen itself
	}
	return info, rel, true
}

func (c *checker) checkModule() {
	m := c.e.Module
	c.refs(refModule, m.DependsOn, "/dependsOn", "dependsOn")
	for i, ep := range m.Endpoints {
		c.refs(refRole, ep.Roles, fmt.Sprintf("/endpoints/%d/roles", i), "endpoints")
	}
}

// checkSources verifies that listed sources exist under the repository root.
// A missing source is an error for unreviewed content and makes approved
// content outdated instead (see the review package).
func (c *checker) checkSources() {
	for i, src := range c.e.Sources {
		p := fmt.Sprintf("/sources/%d", i)
		abs, err := c.ix.SourcePath(src)
		if err != nil {
			c.errorf(p, "%v", err)
			continue
		}
		if c.e.Review.Status == ReviewApproved {
			continue
		}
		st, err := os.Stat(abs)
		switch {
		case err != nil:
			c.errorf(p, "source %s not found under the repository root", src)
		case strings.HasSuffix(src, "/") && !st.IsDir():
			c.errorf(p, "source %s ends in / but is a file", src)
		case !strings.HasSuffix(src, "/") && st.IsDir():
			c.errorf(p, "source %s is a directory; write it as %s/", src, src)
		}
	}
}

// SourcePath resolves a source path against the repository root and rejects
// paths that escape it.
func (ix *Index) SourcePath(src string) (string, error) {
	clean := strings.TrimSuffix(src, "/")
	for _, seg := range strings.Split(clean, "/") {
		if seg == ".." {
			return "", fmt.Errorf("source %s must not contain .. segments", src)
		}
	}
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("source %s must be relative to the repository root", src)
	}
	return filepath.Join(ix.RepoRoot, filepath.FromSlash(clean)), nil
}

func joinRel(dir, p string) string { return dir + "/" + p }
