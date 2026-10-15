package content_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/testutil"
)

func TestDemoIsValid(t *testing.T) {
	ix, err := content.Load(filepath.Join(testutil.RepoRoot(), "examples", "demo", "muninn"))
	if err != nil {
		t.Fatal(err)
	}
	for _, is := range ix.Issues {
		t.Errorf("unexpected issue: %s", is)
	}
	if got := len(ix.All()); got != 12 {
		t.Errorf("entities = %d, want 12", got)
	}
}

// Each rule gets one broken fixture made from the demo.
func TestValidationRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, dir string)
		want   string // substring of an error message
	}{
		{"missing required field", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "features/approvals/feature.md"), "area: approvals\n", "")
		}, "missing property 'area'"},
		{"unknown field", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "features/approvals/feature.md"), "area: approvals\n", "area: approvals\ncolour: red\n")
		}, "additional properties 'colour'"},
		{"generator may not invent review states", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "review: { status: draft }", "review: { status: done }")
		}, "value must be one of"},
		{"approved without the fields muninn writes", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "review: { status: draft }", "review: { status: approved }")
		}, "missing properties 'by', 'at', 'contentHash'"},
		{"changes requested without a note", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "review: { status: draft }", "review: { status: changes_requested, by: A, at: \"2026-10-14T00:00:00Z\" }")
		}, "missing property 'note'"},
		{"unknown feature reference", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "feature: sign-in", "feature: log-in")
		}, `unknown feature "log-in"`},
		{"unknown screen in feature", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "features/sign-in/feature.md"), "screens: [sign-in]", "screens: [sign-in, sign-up]")
		}, `unknown screen "sign-up"`},
		{"unknown hotspot target", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "to: claim-list", "to: home")
		}, `unknown screen "home"`},
		{"unknown role", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "role: employee", "role: manager")
		}, `unknown role "manager"`},
		{"unknown area", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "features/sign-in/feature.md"), "area: account", "area: profile")
		}, `unknown area "profile"`},
		{"unknown term", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "features/approvals/feature.md"), "terms: [claim, return]", "terms: [claim, refund]")
		}, `unknown term "refund"`},
		{"unknown module", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/web-claims.md"), "dependsOn: [app-data]", "dependsOn: [api]")
		}, `unknown module "api"`},
		{"unknown owner area", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "muninn.yaml"), "approvals: [Liam]", "billing: [Liam]")
		}, `unknown area "billing"`},
		{"missing image", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "screens/sign-in/shots/default.webp")); err != nil {
				t.Fatal(err)
			}
		}, "image screens/sign-in/shots/default.webp not found"},
		{"hotspot outside image", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "box: [528, 197, 310, 36]", "box: [1200, 197, 310, 36]")
		}, "lies outside screens/sign-in/shots/default.webp (1366x768)"},
		{"flow box outside reused screen image", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "box: [969, 124, 107, 38]", "box: [969, 750, 107, 38]")
		}, "lies outside screens/claim-list/shots/default.webp"},
		{"unknown variant", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "variant: default", "variant: loading")
		}, `has no variant "loading"`},
		{"duplicate id", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/web-approvals.md"), "id: web-approvals", "id: web-claims")
		}, `duplicate module id "web-claims"`},
		{"id differs from directory", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "id: sign-in", "id: login")
		}, `id "login" must equal the directory name "sign-in"`},
		{"duplicate glossary id", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "glossary.yaml"), "- id: receipt", "- id: claim")
		}, `duplicate id "claim"`},
		{"step with screen and run", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "    screen: claim-list\n", "    screen: claim-list\n    run: { cmd: make }\n")
		}, "Exactly one of screen"},
		{"step with neither screen, shot nor run", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "    shot: shots/step-4.webp\n", "")
		}, "Exactly one of screen"},
		{"run step with box", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "    shot: shots/step-4.webp\n", "    run: { cmd: make worker }\n")
		}, "Exactly one of screen"},
		{"type step without sample", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "    sample:\n", "    examples:\n")
		}, "missing property 'sample'"},
		{"steps out of order", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "flows/submit-claim/flow.md"), "  - n: 3", "  - n: 5")
		}, "step number 5 out of order"},
		{"translation without main file", func(t *testing.T, dir string) {
			testutil.Write(t, filepath.Join(dir, "modules/billing.ja.md"), "## 責務\n")
		}, "translation has no main file modules/billing.md"},
		{"translation with frontmatter", func(t *testing.T, dir string) {
			testutil.Write(t, filepath.Join(dir, "screens/claim-list/screen.ja.md"), "---\nid: claim-list\n---\nbody\n")
		}, "translation files contain a Markdown body only"},
		{"source missing on a draft", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "sources: [app/app.js]", "sources: [app/store.js]")
		}, "source app/store.js not found"},
		{"source escaping the repository", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "sources: [app/app.js]", "sources: [app/../../etc/passwd]")
		}, "must not contain .. segments"},
		{"directory source without slash", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "modules/app-data.md"), "sources: [app/app.js]", "sources: [app]")
		}, "is a directory; write it as app/"},
		{"duplicate hotspot number", func(t *testing.T, dir string) {
			testutil.Edit(t, filepath.Join(dir, "screens/sign-in/screen.md"), "      - n: 2", "      - n: 1")
		}, "duplicate hotspot number 1"},
		{"no frontmatter", func(t *testing.T, dir string) {
			testutil.Write(t, filepath.Join(dir, "modules/app-data.md"), "## Responsibility\n")
		}, "must start with a YAML frontmatter"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, dir := testutil.Demo(t)
			tc.mutate(t, dir)
			ix, err := content.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if ix.ErrorCount() == 0 {
				t.Fatalf("no errors, want %q", tc.want)
			}
			for _, is := range ix.Issues {
				if is.Severity == content.SevError && strings.Contains(is.Message, tc.want) {
					return
				}
			}
			t.Errorf("no error containing %q; got:\n%v", tc.want, ix.Issues)
		})
	}
}

func TestWarnings(t *testing.T) {
	_, dir := testutil.Demo(t)
	big := make([]byte, content.LargeImageBytes+1)
	src, _ := os.ReadFile(filepath.Join(dir, "screens/sign-in/shots/default.webp"))
	copy(big, src) // valid header, padded past the limit
	if err := os.WriteFile(filepath.Join(dir, "screens/sign-in/shots/default.webp"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := content.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ix.ErrorCount() != 0 {
		t.Fatalf("warnings must not be errors: %v", ix.Issues)
	}
	found := false
	for _, is := range ix.Issues {
		found = found || (is.Severity == content.SevWarning && strings.Contains(is.Message, "keep screenshots under 400 KB"))
	}
	if !found {
		t.Errorf("no large image warning: %v", ix.Issues)
	}
}

func TestBackReferences(t *testing.T) {
	ix, err := content.Load(filepath.Join(testutil.RepoRoot(), "examples", "demo", "muninn"))
	if err != nil {
		t.Fatal(err)
	}
	var fromFlow bool
	for _, r := range ix.RefsTo["screen/claim-list"] {
		fromFlow = fromFlow || (r.Kind == content.KindFlow && r.ID == "submit-claim")
	}
	if !fromFlow {
		t.Errorf("claim-list should be referenced by flow submit-claim: %v", ix.RefsTo["screen/claim-list"])
	}
}
