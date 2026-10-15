package review

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/lamngockhuong/muninn/internal/content"
)

// ChangedFiles lists tracked files under root that differ from ref,
// including uncommitted changes to them. Paths are relative to root.
// Requires git.
func ChangedFiles(root, ref string) ([]string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, errors.New("--changed-since needs git installed and on PATH")
	}
	if ref == "" || strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("invalid git ref %q", ref)
	}
	cmd := exec.Command("git", "diff", "--name-only", "--relative", "--no-renames", "--end-of-options", ref, "--", ".")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff %s: %s", ref, strings.TrimSpace(stderr.String()))
	}
	var files []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

// TouchedSources returns the changed files that belong to the entity's
// sources: a file source matches itself, a directory source any file inside.
func TouchedSources(e *content.Entity, changed []string) []string {
	var out []string
	for _, src := range e.Sources {
		for _, f := range changed {
			if f == src {
				out = append(out, src)
			} else if strings.HasSuffix(src, "/") && strings.HasPrefix(f, src) {
				out = append(out, f)
			}
		}
	}
	return out
}
