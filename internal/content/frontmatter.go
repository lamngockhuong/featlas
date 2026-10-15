package content

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is a Markdown file split into its YAML frontmatter and body.
type Document struct {
	Src      []byte // whole file
	FMStart  int    // byte offset of the first frontmatter line
	FMEnd    int    // byte offset of the closing "---" line
	BodyFrom int    // byte offset where the body starts
	Node     *yaml.Node
}

var errNoFrontmatter = errors.New("file must start with a YAML frontmatter block delimited by --- lines")

// utf8BOM is tolerated at the start of a file and kept when it is rewritten.
var utf8BOM = []byte("\xef\xbb\xbf")

// Line breaks that YAML counts but this package does not. Rejecting them
// keeps YAML line numbers and byte offsets in step, which the review block
// rewrite relies on.
var foreignBreaks = []string{"\u2028", "\u2029", "\u0085"}

// ParseDocument splits src into frontmatter and body and parses the
// frontmatter as a YAML mapping.
func ParseDocument(src []byte) (*Document, error) {
	start := 0
	if bytes.HasPrefix(src, utf8BOM) {
		start = len(utf8BOM)
	}
	first, rest, ok := cutLine(src, start)
	if !ok || trimCR(src[start:first]) != "---" {
		return nil, errNoFrontmatter
	}
	pos := rest
	for pos < len(src) {
		end, next, _ := cutLine(src, pos)
		if trimCR(src[pos:end]) == "---" {
			fm := string(src[rest:pos])
			for _, br := range foreignBreaks {
				if strings.Contains(fm, br) {
					return nil, fmt.Errorf("frontmatter contains the line separator %U; use \\n line breaks", []rune(br)[0])
				}
			}
			if strings.Contains(strings.ReplaceAll(fm, "\r\n", ""), "\r") {
				return nil, errors.New("frontmatter contains a bare carriage return; use \\n or \\r\\n line breaks")
			}
			doc := &Document{Src: src, FMStart: rest, FMEnd: pos, BodyFrom: next}
			node, err := parseMapping(src[rest:pos])
			if err != nil {
				return nil, err
			}
			doc.Node = node
			return doc, nil
		}
		pos = next
	}
	return nil, errors.New("frontmatter is not closed by a --- line")
}

// HasFrontmatter reports whether src starts with a "---" line.
func HasFrontmatter(src []byte) bool {
	src = bytes.TrimPrefix(src, utf8BOM)
	end, _, ok := cutLine(src, 0)
	return ok && trimCR(src[:end]) == "---"
}

func (d *Document) Body() string { return string(d.Src[d.BodyFrom:]) }

// cutLine returns the end of the line starting at pos (excluding the newline)
// and the offset of the next line.
func cutLine(src []byte, pos int) (end, next int, ok bool) {
	if pos >= len(src) {
		return pos, pos, false
	}
	i := bytes.IndexByte(src[pos:], '\n')
	if i < 0 {
		return len(src), len(src), true
	}
	return pos + i, pos + i + 1, true
}

func trimCR(b []byte) string { return strings.TrimSuffix(string(b), "\r") }

func parseMapping(b []byte) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	if len(root.Content) == 0 {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}, nil
	}
	n := root.Content[0]
	if n.Kind != yaml.MappingNode {
		return nil, errors.New("frontmatter must be a YAML mapping")
	}
	return n, nil
}

// ParseYAML parses a whole YAML file and returns its root value node.
func ParseYAML(b []byte) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}, nil
	}
	return root.Content[0], nil
}

// ReviewSpan locates the top-level review block of the frontmatter as a byte
// range [Start, End) of whole lines in Src. When the key is absent, Found is
// false and Start == End == FMEnd, the insertion point.
type ReviewSpan struct {
	Start, End int
	Found      bool
	Invalid    bool // the block could not be located safely
}

func (d *Document) ReviewSpan() ReviewSpan {
	lines := lineOffsets(d.Src, d.FMStart, d.FMEnd)
	keyLine, nextLine := -1, len(lines) // 0-based indexes into lines
	if d.Node.Style&yaml.FlowStyle != 0 {
		return ReviewSpan{Start: d.FMEnd, End: d.FMEnd, Invalid: true}
	}
	for i := 0; i+1 < len(d.Node.Content); i += 2 {
		k := d.Node.Content[i]
		if k.Column != 1 {
			// Only block mappings starting at column 1 have line-shaped keys.
			return ReviewSpan{Start: d.FMEnd, End: d.FMEnd, Invalid: true}
		}
		if keyLine >= 0 {
			nextLine = k.Line - 1
			break
		}
		if k.Value == "review" {
			keyLine = k.Line - 1
		}
	}
	if keyLine < 0 {
		return ReviewSpan{Start: d.FMEnd, End: d.FMEnd}
	}
	if keyLine >= len(lines) || nextLine > len(lines) {
		// Line numbers and offsets disagree; refuse to guess a range.
		return ReviewSpan{Start: d.FMEnd, End: d.FMEnd, Invalid: true}
	}
	// Leave unindented comments and blank lines that close the block to the
	// surrounding document.
	for nextLine-1 > keyLine {
		l := strings.TrimRight(string(d.Src[lines[nextLine-1]:lineEnd(lines, nextLine-1, d.FMEnd)]), "\r\n")
		if l == "" || strings.HasPrefix(l, "#") {
			nextLine--
			continue
		}
		break
	}
	end := d.FMEnd
	if nextLine < len(lines) {
		end = lines[nextLine]
	}
	return ReviewSpan{Start: lines[keyLine], End: end, Found: true}
}

func lineOffsets(src []byte, from, to int) []int {
	var out []int
	for pos := from; pos < to; {
		out = append(out, pos)
		_, next, _ := cutLine(src, pos)
		pos = next
	}
	return out
}

func lineEnd(lines []int, i, fallback int) int {
	if i+1 < len(lines) {
		return lines[i+1]
	}
	return fallback
}

// WithoutReview returns Src with the review block removed. Used for hashing,
// so reviewing a file never changes its content hash.
func (d *Document) WithoutReview() []byte {
	s := d.ReviewSpan()
	out := make([]byte, 0, len(d.Src)-(s.End-s.Start))
	out = append(out, d.Src[:s.Start]...)
	return append(out, d.Src[s.End:]...)
}

// Value converts a YAML node into a JSON-compatible value: map[string]any,
// []any, string, int64, float64, bool or nil. Timestamps stay strings, as
// written, so they validate as JSON Schema strings.
func Value(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return Value(n.Content[0])
	case yaml.AliasNode:
		return Value(n.Alias)
	case yaml.MappingNode:
		m := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Tag == "!!merge" {
				return nil, fmt.Errorf("line %d: YAML merge keys are not supported", k.Line)
			}
			val, err := Value(v)
			if err != nil {
				return nil, err
			}
			if _, dup := m[k.Value]; dup {
				return nil, fmt.Errorf("line %d: duplicate key %q", k.Line, k.Value)
			}
			m[k.Value] = val
		}
		return m, nil
	case yaml.SequenceNode:
		s := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			val, err := Value(c)
			if err != nil {
				return nil, err
			}
			s = append(s, val)
		}
		return s, nil
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!null":
			return nil, nil
		case "!!bool":
			var b bool
			err := n.Decode(&b)
			return b, err
		case "!!int":
			if i, err := strconv.ParseInt(n.Value, 0, 64); err == nil {
				return i, nil
			}
			var i int64
			err := n.Decode(&i)
			return i, err
		case "!!float":
			var f float64
			if err := n.Decode(&f); err != nil {
				return nil, err
			}
			if math.IsInf(f, 0) || math.IsNaN(f) {
				return nil, fmt.Errorf("line %d: %s is not a JSON number", n.Line, n.Value)
			}
			return f, nil
		default:
			return n.Value, nil
		}
	}
	return nil, fmt.Errorf("line %d: unsupported YAML node", n.Line)
}
