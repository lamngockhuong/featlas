package review_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/review"
	"github.com/lamngockhuong/muninn/internal/testutil"
)

var fixedReview = content.Review{
	Status:       "approved",
	By:           "Alice",
	At:           "2026-10-14T09:12:00Z",
	ContentHash:  "sha256:" + strings.Repeat("a", 64),
	SourceHashes: map[string]string{"web/b.ts": strings.Repeat("2", 40), "api/a.ts": strings.Repeat("1", 40)},
}

const fixedBlock = `review:
  status: approved
  by: Alice
  at: "2026-10-14T09:12:00Z"
  contentHash: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  sourceHashes:
    api/a.ts: "1111111111111111111111111111111111111111"
    web/b.ts: "2222222222222222222222222222222222222222"
`

// The output differs from the input only in the review block: comments, key
// order, quoting and the body survive byte for byte.
func TestSpliceGolden(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{
			name: "flow review in the middle, comments kept",
			in:   "---\n# screen of the claims area\nid: claim-list   # kept\ntitle: { en: 'Claims' }\nreview: { status: draft }\n\n# sources come last\nsources:\n  - a.ts\n---\n## Body\n\nreview: not frontmatter\n",
			want: "---\n# screen of the claims area\nid: claim-list   # kept\ntitle: { en: 'Claims' }\n" + fixedBlock +
				"\n# sources come last\nsources:\n  - a.ts\n---\n## Body\n\nreview: not frontmatter\n",
		},
		{
			name: "block review at the end",
			in:   "---\nid: x\nreview:\n  status: changes_requested\n  by: Bob\n  note: |\n    fix the\n    amount rule\n---\nbody\n",
			want: "---\nid: x\n" + fixedBlock + "---\nbody\n",
		},
		{
			name: "no review block",
			in:   "---\nid: x\ntitle: { en: X }\n---\n",
			want: "---\nid: x\ntitle: { en: X }\n" + fixedBlock + "---\n",
		},
		{
			name: "CRLF line endings",
			in:   "---\r\nid: x\r\nreview: { status: draft }\r\n---\r\nbody\r\n",
			want: "---\r\nid: x\r\n" + strings.ReplaceAll(fixedBlock, "\n", "\r\n") + "---\r\nbody\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := content.ParseDocument([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			got, err := review.Splice(doc, fixedReview)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
			// Hashing ignores the review block, so the hash survives a review.
			again, err := content.ParseDocument(got)
			if err != nil {
				t.Fatal(err)
			}
			if string(again.WithoutReview()) != string(doc.WithoutReview()) {
				t.Errorf("content without review changed:\n%q\n%q", again.WithoutReview(), doc.WithoutReview())
			}
		})
	}
}

func TestBlobSHAMatchesGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	files := map[string]string{"empty.txt": "", "hello.txt": "hello\n", "utf8.md": "経費申請\r\nno newline at end"}
	for name, data := range files {
		p := filepath.Join(dir, name)
		testutil.Write(t, p, data)
		out, err := exec.Command("git", "hash-object", p).Output()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := review.BlobSHA([]byte(data)), strings.TrimSpace(string(out)); got != want {
			t.Errorf("%s: BlobSHA = %s, git = %s", name, got, want)
		}
	}
}

func TestDirectoryHashHonoursGitignore(t *testing.T) {
	root := t.TempDir()
	testutil.Write(t, filepath.Join(root, ".gitignore"), "*.log\ncache/\n")
	testutil.Write(t, filepath.Join(root, "src/a.ts"), "a")
	h := review.NewHasher(root)
	before, ok, err := h.Source("src/")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	testutil.Write(t, filepath.Join(root, "src/debug.log"), "noise")
	testutil.Write(t, filepath.Join(root, "src/cache/x"), "noise")
	if after, _, _ := h.Source("src/"); after != before {
		t.Error("ignored files changed the directory hash")
	}
	testutil.Write(t, filepath.Join(root, "src/b.ts"), "b")
	if after, _, _ := h.Source("src/"); after == before {
		t.Error("a new file did not change the directory hash")
	}
	if _, ok, _ := h.Source("src"); ok {
		t.Error("a directory written without / must not count as existing")
	}
}

func approve(t *testing.T, dir string, kind content.Kind, id string) {
	t.Helper()
	ix, err := content.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := review.NewHasher(ix.RepoRoot)
	if _, err := review.Apply(ix, h, ix.Get(kind, id), review.Decision{Status: "approved", By: "Alice"}); err != nil {
		t.Fatal(err)
	}
}

func state(t *testing.T, dir string, kind content.Kind, id string) review.Status {
	t.Helper()
	ix, err := content.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if errs := ix.IssuesFor(content.RefKey(kind, id)); len(errs) > 0 {
		t.Fatalf("validation issues: %v", errs)
	}
	return review.Compute(ix, review.NewHasher(ix.RepoRoot), ix.Get(kind, id))
}

func TestComputedStates(t *testing.T) {
	cases := []struct {
		name   string
		kind   content.Kind
		id     string
		change func(t *testing.T, project, dir string)
		want   string
		reason string
	}{
		{"approved stays approved", content.KindScreen, "claim-list", func(*testing.T, string, string) {}, review.StateApproved, ""},
		{"one byte of a source", content.KindScreen, "claim-list", func(t *testing.T, project, _ string) {
			testutil.Edit(t, filepath.Join(project, "app/claims.html"), "My claims", "My Claims")
		}, review.StateOutdated, "source-changed"},
		{"deleted source", content.KindModule, "app-data", func(t *testing.T, project, _ string) {
			if err := os.Remove(filepath.Join(project, "app/app.js")); err != nil {
				t.Fatal(err)
			}
		}, review.StateOutdated, "source-missing"},
		{"edited body", content.KindScreen, "claim-list", func(t *testing.T, _, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/claim-list/screen.md"), "## Purpose", "## Why")
		}, review.StateModified, "content"},
		{"edited translation", content.KindScreen, "claim-list", func(t *testing.T, _, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/claim-list/screen.vi.md"), "Mục đích", "Mục tiêu")
		}, review.StateModified, "content"},
		{"replaced image", content.KindScreen, "claim-list", func(t *testing.T, _, dir string) {
			src, _ := os.ReadFile(filepath.Join(dir, "screens/claim-list/shots/empty.webp"))
			testutil.Write(t, filepath.Join(dir, "screens/claim-list/shots/default.webp"), string(src))
		}, review.StateModified, "content"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, dir := testutil.Demo(t)
			approve(t, dir, tc.kind, tc.id)
			tc.change(t, project, dir)
			st := state(t, dir, tc.kind, tc.id)
			if st.State != tc.want {
				t.Fatalf("state = %s (%v), want %s", st.State, st.Reasons, tc.want)
			}
			if tc.reason != "" && (len(st.Reasons) == 0 || st.Reasons[0].Kind != tc.reason) {
				t.Errorf("reasons = %v, want %s first", st.Reasons, tc.reason)
			}
		})
	}
}

func TestApplyChangesOnlyTheReviewBlock(t *testing.T) {
	_, dir := testutil.Demo(t)
	path := filepath.Join(dir, "screens/claim-list/screen.md")
	before, _ := os.ReadFile(path)
	ix, _ := content.Load(dir)
	h := review.NewHasher(ix.RepoRoot)
	at := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	rv, err := review.Apply(ix, h, ix.Get(content.KindScreen, "claim-list"), review.Decision{Status: "changes_requested", By: "  Alice ", Note: "Row click opens edit, not detail", At: at})
	if err != nil {
		t.Fatal(err)
	}
	if rv.By != "Alice" || rv.ContentHash != "" {
		t.Errorf("review = %+v", rv)
	}
	after, _ := os.ReadFile(path)
	want := strings.Replace(string(before), "review: { status: draft }\n",
		"review:\n  status: changes_requested\n  by: Alice\n  at: \"2026-10-14T09:00:00Z\"\n  note: Row click opens edit, not detail\n", 1)
	if string(after) != want {
		t.Errorf("file differs outside the review block:\n%s", after)
	}
}

func TestDecisionRules(t *testing.T) {
	_, dir := testutil.Demo(t)
	ix, _ := content.Load(dir)
	h := review.NewHasher(ix.RepoRoot)
	e := ix.Get(content.KindModule, "app-data")
	for _, d := range []review.Decision{
		{Status: "approved", By: "   "},
		{Status: "approved", By: strings.Repeat("x", 101)},
		{Status: "draft", By: "Alice"},
		{Status: "changes_requested", By: "Alice"},
	} {
		if _, err := review.Apply(ix, h, e, d); err == nil {
			t.Errorf("%+v accepted", d)
		}
	}
}

// Frontmatter the line-based rewrite cannot handle is refused, never
// corrupted.
func TestSpliceRefusesUnsafeFrontmatter(t *testing.T) {
	for name, in := range map[string]string{
		"unicode line separator": "---\nid: x\ntitle: { en: \"a b\" }\nreview: { status: draft }\ndependsOn: [y]\n---\n",
		"bare carriage return":   "---\nid: x\rtitle: t\nreview: { status: draft }\n---\n",
		"flow mapping root":      "---\n{id: x, review: {status: draft}}\n---\n",
		"one-line flow root":     "---\n{id: x}\n---\n",
		"indented root":          "---\n  id: x\n  review: { status: draft }\n---\n",
		"anchor used elsewhere":  "---\nid: x\nreview: &r { status: draft }\nold: *r\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := content.ParseDocument([]byte(in))
			if err != nil {
				return // rejected while parsing: fine
			}
			if out, err := review.Splice(doc, fixedReview); err == nil {
				t.Errorf("accepted, wrote:\n%s", out)
			}
		})
	}
}

func TestSpliceKeepsBOM(t *testing.T) {
	in := "\xef\xbb\xbf---\nid: x\nreview: { status: draft }\n---\nbody\n"
	doc, err := content.ParseDocument([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := review.Splice(doc, fixedReview)
	if err != nil {
		t.Fatal(err)
	}
	if want := "\xef\xbb\xbf---\nid: x\n" + fixedBlock + "---\nbody\n"; string(out) != want {
		t.Errorf("got %q", out)
	}
}

func TestDirectoryHashFollowsGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	testutil.Write(t, filepath.Join(root, "src/a.ts"), "a")
	testutil.Write(t, filepath.Join(root, "src/gen/.gitignore"), "*.js\n") // nested: only git sees it
	h := review.NewHasher(root)
	before, ok, err := h.Source("src/")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	testutil.Write(t, filepath.Join(root, "src/gen/out.js"), "noise")
	if after, _, _ := review.NewHasher(root).Source("src/"); after != before {
		t.Error("a file ignored by a nested .gitignore changed the hash")
	}
	if _, ok, _ := h.Source("src/../../etc/"); ok {
		t.Error("a source outside the repository must not be hashed")
	}
}
