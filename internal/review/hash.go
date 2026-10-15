// Package review computes content and source hashes, derives the displayed
// review state, and writes the review block of an entity file.
package review

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/lamngockhuong/muninn/internal/content"
)

// BlobSHA returns the git blob SHA-1 of data: sha1("blob <len>\0" + data).
func BlobSHA(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Hasher hashes source files under one repository root. It caches file
// hashes by size and modification time, so repeated index rebuilds only
// re-read changed files.
type Hasher struct {
	root    string
	git     bool // root is inside a git work tree and git is installed
	ignore  *ignore.GitIgnore
	mu      sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	size, mod int64
	sha       string
}

// NewHasher creates a hasher honouring the root .gitignore.
func NewHasher(root string) *Hasher {
	h := &Hasher{root: root, entries: map[string]cacheEntry{}}
	if out, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
		h.git = true
	}
	if gi, err := ignore.CompileIgnoreFile(filepath.Join(root, ".gitignore")); err == nil {
		h.ignore = gi
	}
	return h
}

// safeRel rejects source paths that would leave the repository root.
func safeRel(rel string) bool {
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\\") {
		return false
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// Root is the repository root the hasher resolves paths against.
func (h *Hasher) Root() string { return h.root }

// Source hashes one source path. Files hash to their blob SHA; directories
// (written with a trailing /) hash to the SHA-1 of the sorted "path:blob"
// lines of every non-ignored file inside. exists is false when the path is
// missing or has the wrong type.
func (h *Hasher) Source(src string) (sha string, exists bool, err error) {
	rel := strings.TrimSuffix(src, "/")
	if !safeRel(rel) {
		return "", false, nil
	}
	abs := filepath.Join(h.root, filepath.FromSlash(rel))
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if strings.HasSuffix(src, "/") != st.IsDir() {
		return "", false, nil
	}
	if !st.IsDir() {
		sha, err := h.file(abs, st)
		return sha, err == nil, err
	}
	if h.git {
		return h.gitDir(rel)
	}
	var lines []string
	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relp, _ := filepath.Rel(h.root, p)
		relp = filepath.ToSlash(relp)
		if d.IsDir() {
			if d.Name() == ".git" || (p != abs && h.ignored(relp+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || h.ignored(relp) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		sha, err := h.file(p, info)
		if err != nil {
			return err
		}
		lines = append(lines, relp+":"+sha)
		return nil
	})
	if err != nil {
		return "", false, err
	}
	sort.Strings(lines)
	sum := sha1.Sum([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), true, nil
}

// gitDir hashes the files git sees in a directory: tracked and untracked
// files minus everything ignored by any .gitignore, .git/info/exclude or the
// global excludes. Local checkouts and CI then agree on the file list.
func (h *Hasher) gitDir(rel string) (string, bool, error) {
	out, err := exec.Command("git", "-C", h.root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", rel+"/").Output()
	if err != nil {
		return "", false, fmt.Errorf("git ls-files %s: %w", rel, err)
	}
	seen := map[string]bool{}
	var lines []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		abs := filepath.Join(h.root, filepath.FromSlash(p))
		st, err := os.Lstat(abs)
		if err != nil || !st.Mode().IsRegular() {
			continue // deleted but still tracked, or a symlink
		}
		sha, err := h.file(abs, st)
		if err != nil {
			return "", false, err
		}
		lines = append(lines, p+":"+sha)
	}
	sort.Strings(lines)
	sum := sha1.Sum([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), true, nil
}

func (h *Hasher) ignored(rel string) bool {
	return h.ignore != nil && h.ignore.MatchesPath(rel)
}

func (h *Hasher) file(abs string, st fs.FileInfo) (string, error) {
	h.mu.Lock()
	ce, ok := h.entries[abs]
	h.mu.Unlock()
	if ok && ce.size == st.Size() && ce.mod == st.ModTime().UnixNano() {
		return ce.sha, nil
	}
	// A file written within the timestamp resolution of the filesystem can
	// change again without a new mtime, so recent files are not cached.
	recent := time.Since(st.ModTime()) < 2*time.Second
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	sha := BlobSHA(data)
	if !recent {
		h.mu.Lock()
		h.entries[abs] = cacheEntry{size: st.Size(), mod: st.ModTime().UnixNano(), sha: sha}
		h.mu.Unlock()
	}
	return sha, nil
}

// Sources hashes every source of an entity. Missing sources are left out.
func (h *Hasher) Sources(srcs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, s := range srcs {
		sha, ok, err := h.Source(s)
		if err != nil {
			return nil, err
		}
		if ok {
			out[s] = sha
		}
	}
	return out, nil
}

// ContentHash is sha256 over the entity's files in a fixed order: the main
// file without its review block, translation files by language, then
// referenced images by path. Each part is framed by its path and length so
// moving bytes between files changes the hash.
func ContentHash(ix *content.Index, e *content.Entity) (string, error) {
	if e.Doc == nil {
		return "", fmt.Errorf("%s cannot be parsed", e.File)
	}
	h := sha256.New()
	part := func(name string, data []byte) {
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(data))
		h.Write(data)
	}
	part(e.File, e.Doc.WithoutReview())
	langs := make([]string, 0, len(e.Translations))
	for l := range e.Translations {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	for _, l := range langs {
		data, err := os.ReadFile(ix.Abs(e.Translations[l]))
		if err != nil {
			return "", err
		}
		part(e.Translations[l], data)
	}
	imgs := append([]string(nil), e.Images...)
	sort.Strings(imgs)
	for _, img := range imgs {
		f, err := os.Open(ix.Abs(img))
		if err != nil {
			return "", err
		}
		st, err := f.Stat()
		if err != nil {
			f.Close()
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", img, st.Size())
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
