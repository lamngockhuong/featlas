package review

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/lamngockhuong/muninn/internal/content"
)

// MaxNameLen bounds the reviewer name typed in by a person.
const MaxNameLen = 100

// Decision is a person's review action.
type Decision struct {
	Status string // approved | changes_requested
	By     string
	Note   string
	At     time.Time
}

// ErrInvalid marks a decision rejected before touching the file.
var ErrInvalid = errors.New("invalid review")

func (d *Decision) normalize() error {
	d.By = strings.TrimSpace(d.By)
	d.Note = strings.TrimSpace(d.Note)
	switch {
	case d.Status != content.ReviewApproved && d.Status != content.ReviewChangesRequested:
		return fmt.Errorf("%w: status must be approved or changes_requested", ErrInvalid)
	case d.By == "":
		return fmt.Errorf("%w: reviewer name is required", ErrInvalid)
	case utf8.RuneCountInString(d.By) > MaxNameLen:
		return fmt.Errorf("%w: reviewer name is longer than %d characters", ErrInvalid, MaxNameLen)
	case d.Status == content.ReviewChangesRequested && d.Note == "":
		return fmt.Errorf("%w: a note is required when requesting changes", ErrInvalid)
	}
	if d.At.IsZero() {
		d.At = time.Now()
	}
	return nil
}

// Apply records a decision in the entity's main file. Only the review block
// changes; every other byte of the file is kept. The file is re-read from
// disk so edits made after the index was built are hashed, not overwritten.
func Apply(ix *content.Index, h *Hasher, e *content.Entity, d Decision) (content.Review, error) {
	if err := d.normalize(); err != nil {
		return content.Review{}, err
	}
	abs := ix.Abs(e.File)
	src, err := os.ReadFile(abs)
	if err != nil {
		return content.Review{}, err
	}
	doc, err := content.ParseDocument(src)
	if err != nil {
		return content.Review{}, fmt.Errorf("%s: %w", e.File, err)
	}
	rv := content.Review{
		Status: d.Status,
		By:     d.By,
		At:     d.At.UTC().Format(time.RFC3339),
		Note:   d.Note,
	}
	if d.Status == content.ReviewApproved {
		fresh := *e
		fresh.Doc = doc
		if rv.ContentHash, err = ContentHash(ix, &fresh); err != nil {
			return content.Review{}, err
		}
		if rv.SourceHashes, err = h.Sources(e.Sources); err != nil {
			return content.Review{}, err
		}
	}
	out, err := Splice(doc, rv)
	if err != nil {
		return content.Review{}, err
	}
	return rv, writeAtomic(abs, out)
}

// Splice returns the document with its review block replaced by rv, or with
// rv appended to the frontmatter when there is no review block.
func Splice(doc *content.Document, rv content.Review) ([]byte, error) {
	span := doc.ReviewSpan()
	if span.Invalid {
		return nil, errors.New("cannot locate the review block: the frontmatter must be a block-style YAML mapping with top-level keys at column 1")
	}
	block, err := encodeBlock(rv)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(doc.Src[doc.FMStart:doc.FMEnd], []byte("\r\n")) {
		block = bytes.ReplaceAll(block, []byte("\n"), []byte("\r\n"))
	}
	out := make([]byte, 0, len(doc.Src)+len(block))
	out = append(out, doc.Src[:span.Start]...)
	out = append(out, block...)
	out = append(out, doc.Src[span.End:]...)
	if err := verifySplice(doc, out, rv); err != nil {
		return nil, err
	}
	return out, nil
}

// verifySplice re-reads the rewritten file and refuses it unless the review
// block reads back as written and everything else is unchanged. It guards
// against YAML constructs the line-based rewrite cannot handle (anchors used
// elsewhere, document end markers, unusual indentation).
func verifySplice(orig *content.Document, out []byte, rv content.Review) error {
	fail := func(why string) error {
		return fmt.Errorf("refusing to write the review: %s; rewrite the frontmatter as plain block-style YAML", why)
	}
	doc, err := content.ParseDocument(out)
	if err != nil {
		return fail("the result does not parse (" + err.Error() + ")")
	}
	if !bytes.Equal(doc.WithoutReview(), orig.WithoutReview()) {
		return fail("content outside the review block would change")
	}
	var got struct {
		Review content.Review `yaml:"review"`
	}
	if err := doc.Node.Decode(&got); err != nil {
		return fail("the review block does not read back (" + err.Error() + ")")
	}
	if got.Review.Status != rv.Status || got.Review.By != rv.By || got.Review.At != rv.At || got.Review.Note != rv.Note ||
		got.Review.ContentHash != rv.ContentHash || len(got.Review.SourceHashes) != len(rv.SourceHashes) {
		return fail("the review block reads back differently")
	}
	for k, v := range rv.SourceHashes {
		if got.Review.SourceHashes[k] != v {
			return fail("the review block reads back differently")
		}
	}
	return nil
}

// encodeBlock renders "review:" with keys in a fixed order and source hashes
// sorted, so repeated reviews produce minimal diffs.
func encodeBlock(rv content.Review) ([]byte, error) {
	m := &yaml.Node{Kind: yaml.MappingNode}
	add := func(k string, v *yaml.Node) {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, v)
	}
	str := func(s string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s} }
	add("status", str(rv.Status))
	if rv.By != "" {
		add("by", str(rv.By))
	}
	if rv.At != "" {
		add("at", str(rv.At))
	}
	if rv.Note != "" {
		add("note", str(rv.Note))
	}
	if rv.ContentHash != "" {
		add("contentHash", str(rv.ContentHash))
	}
	if len(rv.SourceHashes) > 0 {
		keys := make([]string, 0, len(rv.SourceHashes))
		for k := range rv.SourceHashes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sh := &yaml.Node{Kind: yaml.MappingNode}
		for _, k := range keys {
			sh.Content = append(sh.Content, str(k), str(rv.SourceHashes[k]))
		}
		add("sourceHashes", sh)
	}
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "review"}, m}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeAtomic(path string, data []byte) error {
	// Write through a symlink to its target instead of replacing the link.
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(st.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
